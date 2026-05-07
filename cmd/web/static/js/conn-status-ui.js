(function () {
  var MODE_LABEL = {
    ipv4: 'IPv4 only',
    ipv6: 'IPv6 only',
    dual: 'Dual stack'
  };

  function $(id) {
    return document.getElementById(id);
  }

  function formatAddr(v) {
    if (v == null || v === '') return 'Not available';
    return String(v);
  }

  window.__updateConnStatus = function (state) {
    var btn = $('conn-status-btn');
    var dd4 = $('conn-status-ipv4');
    var dd6 = $('conn-status-ipv6');
    if (!btn || !dd4 || !dd6) return;

    var mode = state.mode;
    var label = MODE_LABEL[mode] || '—';
    btn.querySelector('.conn-status-btn__label').textContent = label;
    btn.setAttribute('aria-label', 'Your connection: ' + label + '. Open details.');
    btn.removeAttribute('aria-disabled');
    btn.disabled = false;
    btn.setAttribute('aria-busy', 'false');

    btn.classList.remove('conn-status-btn--ipv4', 'conn-status-btn--ipv6', 'conn-status-btn--dual');
    if (mode === 'ipv4') btn.classList.add('conn-status-btn--ipv4');
    if (mode === 'ipv6') btn.classList.add('conn-status-btn--ipv6');
    if (mode === 'dual') btn.classList.add('conn-status-btn--dual');

    dd4.textContent = formatAddr(state.ipv4);
    dd6.textContent = formatAddr(state.ipv6);
  };

  function init() {
    var btn = $('conn-status-btn');
    var dlg = $('conn-status-dialog');
    if (!btn || !dlg) return;

    var closeBtn = dlg.querySelector('.conn-status-dialog__close');
    btn.addEventListener('click', function () {
      if (btn.disabled) return;
      dlg.showModal();
      btn.setAttribute('aria-expanded', 'true');
      if (closeBtn) closeBtn.focus();
    });
    dlg.addEventListener('close', function () {
      btn.setAttribute('aria-expanded', 'false');
      btn.focus();
    });
    if (closeBtn) {
      closeBtn.addEventListener('click', function () {
        dlg.close();
      });
    }
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
