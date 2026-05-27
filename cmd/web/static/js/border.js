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
      .then(function (body) {
        var ok = !!(body && body.ok);
        var ip = body && typeof body.ip === 'string' && body.ip !== '' ? body.ip : null;
        var family =
          body && (body.family === 'ipv4' || body.family === 'ipv6') ? body.family : null;
        return { ok: ok, ip: ip, family: family };
      })
      .catch(function () {
        return { ok: false, ip: null, family: null };
      });
  }

  function preferredFromMeta(meta) {
    if (!meta || !meta.ok || !meta.ip) return null;
    return { ip: meta.ip, family: meta.family };
  }

  function fetchDS(signal) {
    if (!ds) return Promise.resolve(null);
    return Promise.allSettled([
      fetch(ds, { mode: 'cors', signal: signal, credentials: 'omit' }),
    ]).then(function (results) {
      return probeMeta(results[0]);
    });
  }

  var v4 = typeof window.__PROBE_V4__ === 'string' ? window.__PROBE_V4__ : '';
  var v6 = typeof window.__PROBE_V6__ === 'string' ? window.__PROBE_V6__ : '';
  var ds = typeof window.__PROBE_DS__ === 'string' ? window.__PROBE_DS__ : '';

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
        var preferred = preferredFromMeta(dsMeta);
        if (j.family === 'ipv6') {
          applyBorderClass('border--ipv6');
          notify('ipv6', null, ipVal, preferred);
        } else {
          applyBorderClass('border--ipv4');
          notify('ipv4', ipVal, null, preferred);
        }
      })
      .catch(function () {
        clearTimeout(toFallback);
        notify(ssrBorderMode(), null, null, null);
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
      if (ok4 && ok6) {
        applyBorderClass('border--dual');
        notify('dual', ip4, ip6, preferred);
      } else if (ok6) {
        applyBorderClass('border--ipv6');
        notify('ipv6', ip4, ip6, preferred);
      } else if (ok4) {
        applyBorderClass('border--ipv4');
        notify('ipv4', ip4, ip6, preferred);
      } else {
        notify(ssrBorderMode(), ip4, ip6, preferred);
      }
    })
    .catch(function () {
      notify(ssrBorderMode(), null, null, null);
    });
})();
