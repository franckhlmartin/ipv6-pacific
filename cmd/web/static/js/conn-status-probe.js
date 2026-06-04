(function (global) {
  var ns = (global.IPv6PacificConnStatus = global.IPv6PacificConnStatus || {});

  function hasRetryOverIPv6(res) {
    if (!res || !res.headers) return false;
    var h = res.headers.get('Retry-Over-IPv6');
    return h === '?1' || h === '1';
  }

  function isIPv4OutageResponse(res) {
    if (!res) return false;
    if (res.status === 566) return true;
    return res.status === 503 && hasRetryOverIPv6(res);
  }

  function parseHealthzBody(body) {
    var ok = !!(body && body.ok);
    var ip = body && typeof body.ip === 'string' && body.ip !== '' ? body.ip : null;
    var family =
      body && (body.family === 'ipv4' || body.family === 'ipv6') ? body.family : null;
    return { ok: ok, ip: ip, family: family, ipv4Outage: !!(body && body.ipv4Outage) };
  }

  function preferredFromMeta(meta) {
    if (!meta || !meta.ok || !meta.ip) return null;
    return { ip: meta.ip, family: meta.family };
  }

  function probeMeta(settled) {
    if (settled.status !== 'fulfilled') {
      return Promise.resolve({ ok: false, ip: null, family: null });
    }
    var res = settled.value;
    if (isIPv4OutageResponse(res)) {
      return Promise.resolve({ ok: false, ip: null, family: null, ipv4Outage: true });
    }
    if (!res.ok) {
      return Promise.resolve({ ok: false, ip: null, family: null });
    }
    return res
      .json()
      .then(parseHealthzBody)
      .catch(function () {
        return { ok: false, ip: null, family: null };
      });
  }

  function fetchSameOriginHealthz(signal) {
    return fetch('/api/healthz', { credentials: 'same-origin', signal: signal })
      .then(function (res) {
        if (isIPv4OutageResponse(res)) {
          return { ok: false, ip: null, family: null, ipv4Outage: true };
        }
        if (!res.ok) {
          return { ok: false, ip: null, family: null };
        }
        return res.json().then(parseHealthzBody);
      })
      .catch(function () {
        return { ok: false, ip: null, family: null };
      });
  }

  function probeOrigin(url) {
    try {
      return new URL(url).origin;
    } catch (e) {
      return '';
    }
  }

  function isLocalDevPage() {
    var h = global.location.hostname;
    return h === 'localhost' || h === '127.0.0.1' || h === '[::1]';
  }

  function usesCrossOriginProbes(v4, v6, ds) {
    var page = global.location.origin;
    return [v4, v6, ds].some(function (u) {
      return u && probeOrigin(u) !== page;
    });
  }

  function fetchDS(ds, signal) {
    if (!ds) return Promise.resolve(null);
    return Promise.allSettled([
      fetch(ds, { mode: 'cors', signal: signal, credentials: 'omit' }),
    ]).then(function (results) {
      return probeMeta(results[0]);
    });
  }

  function fillPreferredFromSameOrigin(preferred, signal) {
    if (preferred) return Promise.resolve(preferred);
    return fetchSameOriginHealthz(signal).then(preferredFromMeta);
  }

  function notifyResult(onResult, mode, ipv4, ipv6, preferred) {
    if (typeof onResult === 'function') {
      onResult({ mode: mode, ipv4: ipv4, ipv6: ipv6, preferred: preferred });
    }
  }

  function runSameOriginOnly(signal, onResult, onIPv4Outage) {
    return Promise.all([
      fetch('/api/client-ip-family', {
        credentials: 'same-origin',
        signal: signal,
      }).then(function (r) {
        return r.json();
      }),
      fetchSameOriginHealthz(signal),
    ]).then(function (pair) {
      var j = pair[0];
      var hz = pair[1];
      if (hz && hz.ipv4Outage) {
        if (typeof onIPv4Outage === 'function') onIPv4Outage();
        return;
      }
      var preferred = preferredFromMeta(hz);
      var ip = typeof j.ip === 'string' ? j.ip : '';
      var ipVal = ip !== '' ? ip : null;
      if (j.family === 'ipv6') {
        notifyResult(onResult, 'ipv6', null, ipVal, preferred);
      } else {
        notifyResult(onResult, 'ipv4', ipVal, null, preferred);
      }
    });
  }

  function ssrFallbackMode() {
    if (global.document && global.document.body) {
      if (global.document.body.classList.contains('border--dual')) return 'dual';
      if (global.document.body.classList.contains('border--ipv6')) return 'ipv6';
    }
    return 'ipv4';
  }

  /**
   * opts: { v4, v6, ds, sameOriginFallback, onResult, onIPv4Outage, ssrFallback }
   */
  ns.runProbe = function (opts) {
    opts = opts || {};
    var v4 = typeof opts.v4 === 'string' ? opts.v4 : '';
    var v6 = typeof opts.v6 === 'string' ? opts.v6 : '';
    var ds = typeof opts.ds === 'string' ? opts.ds : '';
    var onResult = opts.onResult;
    var onIPv4Outage = opts.onIPv4Outage;
    var sameOriginFallback = opts.sameOriginFallback !== false;
    var fallbackMode =
      typeof opts.ssrFallback === 'function' ? opts.ssrFallback : ssrFallbackMode;

    function recoverFromFailedCrossOriginProbes(signal) {
      if (!sameOriginFallback) {
        notifyResult(onResult, fallbackMode(), null, null, null);
        return Promise.resolve();
      }
      return runSameOriginOnly(signal, onResult, onIPv4Outage).catch(function () {
        notifyResult(onResult, fallbackMode(), null, null, null);
      });
    }

    if (sameOriginFallback && usesCrossOriginProbes(v4, v6, ds) && isLocalDevPage()) {
      var ctlDev = new AbortController();
      var toDev = setTimeout(function () {
        ctlDev.abort();
      }, 4000);
      return runSameOriginOnly(ctlDev.signal, onResult, onIPv4Outage)
        .catch(function () {
          notifyResult(onResult, fallbackMode(), null, null, null);
        })
        .then(function () {
          clearTimeout(toDev);
        });
    }

    if (!v4 || !v6) {
      if (!sameOriginFallback) {
        notifyResult(onResult, fallbackMode(), null, null, null);
        return Promise.resolve();
      }
      var ctlFallback = new AbortController();
      var toFallback = setTimeout(function () {
        ctlFallback.abort();
      }, 4000);

      return Promise.all([
        fetch('/api/client-ip-family', {
          credentials: 'same-origin',
          signal: ctlFallback.signal,
        }).then(function (r) {
          return r.json();
        }),
        fetchDS(ds, ctlFallback.signal),
      ])
        .then(function (pair) {
          clearTimeout(toFallback);
          var j = pair[0];
          var dsMeta = pair[1];
          var ip = typeof j.ip === 'string' ? j.ip : '';
          var ipVal = ip !== '' ? ip : null;
          return fillPreferredFromSameOrigin(
            preferredFromMeta(dsMeta),
            ctlFallback.signal
          ).then(function (preferred) {
            if (preferred && preferred.ipv4Outage) {
              if (typeof onIPv4Outage === 'function') onIPv4Outage();
              return;
            }
            if (j.family === 'ipv6') {
              notifyResult(onResult, 'ipv6', null, ipVal, preferred);
            } else {
              notifyResult(onResult, 'ipv4', ipVal, null, preferred);
            }
          });
        })
        .catch(function () {
          clearTimeout(toFallback);
          return recoverFromFailedCrossOriginProbes(ctlFallback.signal);
        });
    }

    var ctl = new AbortController();
    var to = setTimeout(function () {
      ctl.abort();
    }, 4000);

    var fetches = [
      fetch(v4, { mode: 'cors', signal: ctl.signal, credentials: 'omit' }),
      fetch(v6, { mode: 'cors', signal: ctl.signal, credentials: 'omit' }),
    ];
    if (ds) {
      fetches.push(fetch(ds, { mode: 'cors', signal: ctl.signal, credentials: 'omit' }));
    }

    return Promise.allSettled(fetches)
      .then(function (results) {
        clearTimeout(to);
        var metaPromises = [probeMeta(results[0]), probeMeta(results[1])];
        if (ds) {
          metaPromises.push(probeMeta(results[2]));
        }
        return Promise.all(metaPromises).then(function (metas) {
          return { metas: metas, hasDS: !!ds };
        });
      })
      .then(function (out) {
        var metas = out.metas;
        if (metas.some(function (m) {
          return m && m.ipv4Outage;
        })) {
          if (typeof onIPv4Outage === 'function') onIPv4Outage();
          return;
        }
        var ok4 = metas[0].ok;
        var ok6 = metas[1].ok;
        var ip4 = metas[0].ip;
        var ip6 = metas[1].ip;
        var preferred = out.hasDS ? preferredFromMeta(metas[2]) : null;

        function finish(pref) {
          if (ok4 && ok6) {
            notifyResult(onResult, 'dual', ip4, ip6, pref);
          } else if (ok6) {
            notifyResult(onResult, 'ipv6', ip4, ip6, pref);
          } else if (ok4) {
            notifyResult(onResult, 'ipv4', ip4, ip6, pref);
          } else if (usesCrossOriginProbes(v4, v6, ds)) {
            return recoverFromFailedCrossOriginProbes(ctl.signal);
          } else {
            notifyResult(onResult, fallbackMode(), ip4, ip6, pref);
          }
        }

        if (out.hasDS && !preferred && sameOriginFallback) {
          return fillPreferredFromSameOrigin(null, ctl.signal).then(finish);
        }
        finish(preferred);
      })
      .catch(function () {
        if (usesCrossOriginProbes(v4, v6, ds)) {
          return recoverFromFailedCrossOriginProbes(ctl.signal);
        }
        notifyResult(onResult, fallbackMode(), null, null, null);
      });
  };
})(typeof window !== 'undefined' ? window : this);
