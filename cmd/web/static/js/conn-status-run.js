(function () {
  if (typeof window.IPv6PacificConnStatus === 'undefined') return;

  var root = document.querySelector('.ipv6-pacific-conn');
  if (!root) return;

  var variant = root.getAttribute('data-variant') || 'embed';
  var ui = window.IPv6PacificConnStatus.initUI(root, { variant: variant });

  var v4 = typeof window.__PROBE_V4__ === 'string' ? window.__PROBE_V4__ : '';
  var v6 = typeof window.__PROBE_V6__ === 'string' ? window.__PROBE_V6__ : '';
  var ds = typeof window.__PROBE_DS__ === 'string' ? window.__PROBE_DS__ : '';

  function tryOutageRecovery(state) {
    if (variant !== 'outage566') return;
    var token = root.getAttribute('data-outage-token');
    if (!token || !state || !state.ipv6) return;
    fetch(window.location.pathname + window.location.search, {
      method: 'GET',
      credentials: 'same-origin',
      headers: {
        'Retry-Over-IPv6-Recovery': 'recovered; token="' + token + '"',
      },
    }).catch(function () {});
  }

  window.IPv6PacificConnStatus.runProbe({
    v4: v4,
    v6: v6,
    ds: ds,
    sameOriginFallback: true,
    onResult: function (state) {
      ui.update(state);
      tryOutageRecovery(state);
    },
    onIPv4Outage: function () {
      ui.update({ mode: 'ipv4outage', ipv4: null, ipv6: null, preferred: null });
    },
  });
})();
