(function () {
  function applyBorderClass(cls) {
    document.body.classList.remove('border--ipv4', 'border--ipv6', 'border--dual');
    document.body.classList.add(cls);
  }

  var v4 = typeof window.__PROBE_V4__ === 'string' ? window.__PROBE_V4__ : '';
  var v6 = typeof window.__PROBE_V6__ === 'string' ? window.__PROBE_V6__ : '';

  if (!v4 || !v6) {
    fetch('/api/client-ip-family', { credentials: 'same-origin' })
      .then(function (r) { return r.json(); })
      .then(function (j) {
        if (j.family === 'ipv6') {
          applyBorderClass('border--ipv6');
        } else {
          applyBorderClass('border--ipv4');
        }
      })
      .catch(function () {});
    return;
  }

  var ctl = new AbortController();
  var to = setTimeout(function () { ctl.abort(); }, 4000);
  Promise.allSettled([
    fetch(v4, { mode: 'cors', signal: ctl.signal, credentials: 'omit' }),
    fetch(v6, { mode: 'cors', signal: ctl.signal, credentials: 'omit' }),
  ]).then(function (results) {
    clearTimeout(to);
    var ok4 = results[0].status === 'fulfilled' && results[0].value.ok;
    var ok6 = results[1].status === 'fulfilled' && results[1].value.ok;
    if (ok4 && ok6) {
      applyBorderClass('border--dual');
    } else if (ok6) {
      applyBorderClass('border--ipv6');
    } else if (ok4) {
      applyBorderClass('border--ipv4');
    }
  });
})();
