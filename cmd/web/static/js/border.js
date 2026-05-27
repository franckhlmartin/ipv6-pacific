(function () {
  function applyBorderClass(cls) {
    document.body.classList.remove('border--ipv4', 'border--ipv6', 'border--dual');
    document.body.classList.add(cls);
  }

  function notify(mode, ipv4, ipv6, preferred) {
    if (typeof window.__updateConnStatus === 'function') {
      window.__updateConnStatus({
        mode: mode,
        ipv4: ipv4,
        ipv6: ipv6,
        preferred: preferred,
      });
    }
  }

  function ssrBorderMode() {
    if (document.body.classList.contains('border--dual')) return 'dual';
    if (document.body.classList.contains('border--ipv6')) return 'ipv6';
    return 'ipv4';
  }

  function parseHealthzBody(body) {
    var ok = !!(body && body.ok);
    var ip = body && typeof body.ip === 'string' && body.ip !== '' ? body.ip : null;
    var family =
      body && (body.family === 'ipv4' || body.family === 'ipv6') ? body.family : null;
    return { ok: ok, ip: ip, family: family };
  }

  function probeMeta(settled) {
    if (settled.status !== 'fulfilled') {
      return Promise.resolve({ ok: false, ip: null, family: null });
    }
    var res = settled.value;
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

  function preferredFromMeta(meta) {
    if (!meta || !meta.ok || !meta.ip) return null;
    return { ip: meta.ip, family: meta.family };
  }

  function fetchSameOriginHealthz(signal) {
    return fetch('/api/healthz', { credentials: 'same-origin', signal: signal })
      .then(function (res) {
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
    var h = window.location.hostname;
    return h === 'localhost' || h === '127.0.0.1' || h === '[::1]';
  }

  function usesCrossOriginProbes() {
    var page = window.location.origin;
    return [v4, v6, ds].some(function (u) {
      return u && probeOrigin(u) !== page;
    });
  }

  function fetchDS(signal) {
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

  function runSameOriginOnly(signal) {
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
      var preferred = preferredFromMeta(hz);
      var ip = typeof j.ip === 'string' ? j.ip : '';
      var ipVal = ip !== '' ? ip : null;
      if (j.family === 'ipv6') {
        applyBorderClass('border--ipv6');
        notify('ipv6', null, ipVal, preferred);
      } else {
        applyBorderClass('border--ipv4');
        notify('ipv4', ipVal, null, preferred);
      }
    });
  }

  function recoverFromFailedCrossOriginProbes(signal) {
    return runSameOriginOnly(signal).catch(function () {
      notify(ssrBorderMode(), null, null, null);
    });
  }

  var v4 = typeof window.__PROBE_V4__ === 'string' ? window.__PROBE_V4__ : '';
  var v6 = typeof window.__PROBE_V6__ === 'string' ? window.__PROBE_V6__ : '';
  var ds = typeof window.__PROBE_DS__ === 'string' ? window.__PROBE_DS__ : '';

  if (usesCrossOriginProbes() && isLocalDevPage()) {
    var ctlDev = new AbortController();
    var toDev = setTimeout(function () {
      ctlDev.abort();
    }, 4000);
    runSameOriginOnly(ctlDev.signal)
      .catch(function () {
        notify(ssrBorderMode(), null, null, null);
      })
      .then(function () {
        clearTimeout(toDev);
      });
    return;
  }

  if (!v4 || !v6) {
    var ctlFallback = new AbortController();
    var toFallback = setTimeout(function () {
      ctlFallback.abort();
    }, 4000);

    Promise.all([
      fetch('/api/client-ip-family', {
        credentials: 'same-origin',
        signal: ctlFallback.signal,
      }).then(function (r) {
        return r.json();
      }),
      fetchDS(ctlFallback.signal),
    ])
      .then(function (pair) {
        clearTimeout(toFallback);
        var j = pair[0];
        var dsMeta = pair[1];
        var ip = typeof j.ip === 'string' ? j.ip : '';
        var ipVal = ip !== '' ? ip : null;
        return fillPreferredFromSameOrigin(preferredFromMeta(dsMeta), ctlFallback.signal).then(
          function (preferred) {
            if (j.family === 'ipv6') {
              applyBorderClass('border--ipv6');
              notify('ipv6', null, ipVal, preferred);
            } else {
              applyBorderClass('border--ipv4');
              notify('ipv4', ipVal, null, preferred);
            }
          }
        );
      })
      .catch(function () {
        clearTimeout(toFallback);
        recoverFromFailedCrossOriginProbes(ctlFallback.signal);
      });
    return;
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

  Promise.allSettled(fetches)
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
      var ok4 = metas[0].ok;
      var ok6 = metas[1].ok;
      var ip4 = metas[0].ip;
      var ip6 = metas[1].ip;
      var preferred = out.hasDS ? preferredFromMeta(metas[2]) : null;

      function finish(pref) {
        if (ok4 && ok6) {
          applyBorderClass('border--dual');
          notify('dual', ip4, ip6, pref);
        } else if (ok6) {
          applyBorderClass('border--ipv6');
          notify('ipv6', ip4, ip6, pref);
        } else if (ok4) {
          applyBorderClass('border--ipv4');
          notify('ipv4', ip4, ip6, pref);
        } else if (usesCrossOriginProbes()) {
          recoverFromFailedCrossOriginProbes(ctl.signal);
        } else {
          notify(ssrBorderMode(), ip4, ip6, pref);
        }
      }

      if (out.hasDS && !preferred) {
        return fillPreferredFromSameOrigin(null, ctl.signal).then(finish);
      }
      finish(preferred);
    })
    .catch(function () {
      if (usesCrossOriginProbes()) {
        recoverFromFailedCrossOriginProbes(ctl.signal);
      } else {
        notify(ssrBorderMode(), null, null, null);
      }
    });
})();
