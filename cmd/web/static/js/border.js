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

  function onResult(state) {
    var mode = state.mode;
    if (mode === 'dual') applyBorderClass('border--dual');
    else if (mode === 'ipv6') applyBorderClass('border--ipv6');
    else applyBorderClass('border--ipv4');
    notify(mode, state.ipv4, state.ipv6, state.preferred);
  }

  function onIPv4Outage() {
    applyBorderClass('border--ipv4');
    notify('ipv4outage', null, null, null);
  }

  var v4 = typeof window.__PROBE_V4__ === 'string' ? window.__PROBE_V4__ : '';
  var v6 = typeof window.__PROBE_V6__ === 'string' ? window.__PROBE_V6__ : '';
  var ds = typeof window.__PROBE_DS__ === 'string' ? window.__PROBE_DS__ : '';

  if (typeof window.IPv6PacificConnStatus === 'undefined') return;

  window.IPv6PacificConnStatus.runProbe({
    v4: v4,
    v6: v6,
    ds: ds,
    sameOriginFallback: true,
    onResult: onResult,
    onIPv4Outage: onIPv4Outage,
  });
})();
