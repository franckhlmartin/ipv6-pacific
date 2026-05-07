(function () {
  function applyBorderClass(cls) {
    document.body.classList.remove('border--ipv4', 'border--ipv6', 'border--dual');
    document.body.classList.add(cls);
  }

  function notify(mode, ipv4, ipv6) {
    if (typeof window.__updateConnStatus === 'function') {
      window.__updateConnStatus({ mode: mode, ipv4: ipv4, ipv6: ipv6 });
    }
  }

  function ssrBorderMode() {
    if (document.body.classList.contains('border--dual')) return 'dual';
    if (document.body.classList.contains('border--ipv6')) return 'ipv6';
    return 'ipv4';
  }

  function probeMeta(settled) {
    if (settled.status !== 'fulfilled') {
      return Promise.resolve({ ok: false, ip: null });
    }
    var res = settled.value;
    if (!res.ok) {
      return Promise.resolve({ ok: false, ip: null });
    }
    return res
      .json()
      .then(function (body) {
        var ok = !!(body && body.ok);
        var ip = body && typeof body.ip === 'string' && body.ip !== '' ? body.ip : null;
        return { ok: ok, ip: ip };
      })
      .catch(function () {
        return { ok: false, ip: null };
      });
  }

  var v4 = typeof window.__PROBE_V4__ === 'string' ? window.__PROBE_V4__ : '';
  var v6 = typeof window.__PROBE_V6__ === 'string' ? window.__PROBE_V6__ : '';

  if (!v4 || !v6) {
    fetch('/api/client-ip-family', { credentials: 'same-origin' })
      .then(function (r) {
        return r.json();
      })
      .then(function (j) {
        var ip = typeof j.ip === 'string' ? j.ip : '';
        var ipVal = ip !== '' ? ip : null;
        if (j.family === 'ipv6') {
          applyBorderClass('border--ipv6');
          notify('ipv6', null, ipVal);
        } else {
          applyBorderClass('border--ipv4');
          notify('ipv4', ipVal, null);
        }
      })
      .catch(function () {
        notify(ssrBorderMode(), null, null);
      });
    return;
  }

  var ctl = new AbortController();
  var to = setTimeout(function () {
    ctl.abort();
  }, 4000);
  Promise.allSettled([
    fetch(v4, { mode: 'cors', signal: ctl.signal, credentials: 'omit' }),
    fetch(v6, { mode: 'cors', signal: ctl.signal, credentials: 'omit' }),
  ])
    .then(function (results) {
      clearTimeout(to);
      return Promise.all([probeMeta(results[0]), probeMeta(results[1])]);
    })
    .then(function (metas) {
      var ok4 = metas[0].ok;
      var ok6 = metas[1].ok;
      var ip4 = metas[0].ip;
      var ip6 = metas[1].ip;
      if (ok4 && ok6) {
        applyBorderClass('border--dual');
        notify('dual', ip4, ip6);
      } else if (ok6) {
        applyBorderClass('border--ipv6');
        notify('ipv6', ip4, ip6);
      } else if (ok4) {
        applyBorderClass('border--ipv4');
        notify('ipv4', ip4, ip6);
      } else {
        notify(ssrBorderMode(), ip4, ip6);
      }
    })
    .catch(function () {
      notify(ssrBorderMode(), null, null);
    });
})();
