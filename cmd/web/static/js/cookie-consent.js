(function () {
  var KEY = 'ipv6-pacific-cookie-consent';

  function getStored() {
    try {
      return localStorage.getItem(KEY);
    } catch (e) {
      return null;
    }
  }

  function setStored(v) {
    try {
      localStorage.setItem(KEY, v);
    } catch (e) {}
  }

  function hideBanner() {
    var el = document.getElementById('cookie-banner-root');
    if (el) {
      el.hidden = true;
      el.setAttribute('aria-hidden', 'true');
    }
  }

  function grant() {
    if (typeof gtag === 'function') {
      gtag('consent', 'update', {
        analytics_storage: 'granted',
        ad_storage: 'denied',
        ad_user_data: 'denied',
        ad_personalization: 'denied'
      });
    }
    setStored('accepted');
    hideBanner();
  }

  function deny() {
    if (typeof gtag === 'function') {
      gtag('consent', 'update', {
        analytics_storage: 'denied',
        ad_storage: 'denied',
        ad_user_data: 'denied',
        ad_personalization: 'denied'
      });
    }
    setStored('rejected');
    hideBanner();
  }

  function init() {
    var acc = document.getElementById('cookie-banner-accept');
    var rej = document.getElementById('cookie-banner-reject');
    if (acc) {
      acc.addEventListener('click', function (e) {
        e.preventDefault();
        grant();
      });
    }
    if (rej) {
      rej.addEventListener('click', function (e) {
        e.preventDefault();
        deny();
      });
    }
    document.addEventListener(
      'click',
      function (e) {
        if (getStored() !== null) return;
        if (e.target.closest && (e.target.closest('#cookie-banner-accept') || e.target.closest('#cookie-banner-reject'))) {
          return;
        }
        grant();
      },
      false
    );
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
