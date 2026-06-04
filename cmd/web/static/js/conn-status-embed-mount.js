(function () {
  if (typeof window.IPv6PacificConnStatus === 'undefined') return;

  var script = document.currentScript;
  var mountSel =
    (script && script.getAttribute('data-mount')) || '#ipv6-conn-status';
  var mount = document.querySelector(mountSel);
  if (!mount) return;

  mount.classList.add('ipv6-pacific-conn');
  mount.setAttribute('data-variant', 'embedScript');

  if (!mount.querySelector('.conn-status-btn')) {
    mount.innerHTML =
      '<div class="conn-status-header">' +
      '<button type="button" class="conn-status-btn" aria-expanded="false" aria-haspopup="dialog" aria-busy="true" aria-disabled="true" disabled>' +
      '<span class="conn-status-btn__swatch" aria-hidden="true"></span>' +
      '<span class="conn-status-btn__label">Checking…</span>' +
      '</button></div>' +
      '<dialog class="conn-status-dialog">' +
      '<div class="conn-status-dialog__inner">' +
      '<h2 class="conn-status-dialog__title">Your connection</h2>' +
      '<p class="conn-status-dialog__subtitle">How we categorized your connection to the Pacific IPv6 probe network.</p>' +
      '<p class="conn-status-dialog__note">Addresses reflect what our servers see for each request. With NAT, VPNs, or proxies they may show a client, edge, or shared address—not your device’s only possible address.</p>' +
      '<dl class="conn-status-dialog__addrs">' +
      '<div class="conn-status-dialog__row"><dt>Preferred for this site</dt><dd data-conn-preferred>Not available</dd></div>' +
      '<div class="conn-status-dialog__row"><dt>IPv4</dt><dd data-conn-ipv4>Not available</dd></div>' +
      '<div class="conn-status-dialog__row"><dt>IPv6</dt><dd data-conn-ipv6>Not available</dd></div>' +
      '</dl>' +
      '<p class="conn-status-dialog__embed"><a href="{{SITE_URL}}/embed">Embed this widget on your site</a></p>' +
      '<button type="button" class="conn-status-dialog__close">Close</button>' +
      '</div></dialog>';
  }

  var ui = window.IPv6PacificConnStatus.initUI(mount, { variant: 'embedScript' });

  var v4 = typeof window.__PROBE_V4__ === 'string' ? window.__PROBE_V4__ : '';
  var v6 = typeof window.__PROBE_V6__ === 'string' ? window.__PROBE_V6__ : '';
  var ds = typeof window.__PROBE_DS__ === 'string' ? window.__PROBE_DS__ : '';

  function scriptOrigin() {
    if (!script || !script.src) return '';
    try {
      return new URL(script.src).origin;
    } catch (e) {
      return '';
    }
  }

  var sameOriginFallback =
    (!v4 || !v6) && scriptOrigin() === window.location.origin;

  window.IPv6PacificConnStatus.runProbe({
    v4: v4,
    v6: v6,
    ds: ds,
    sameOriginFallback: sameOriginFallback,
    onResult: function (state) {
      ui.update(state);
    },
    onIPv4Outage: function () {
      ui.update({ mode: 'ipv4outage', ipv4: null, ipv6: null, preferred: null });
    },
  });
})();
