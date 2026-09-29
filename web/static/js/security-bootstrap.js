(function () {
    'use strict';

    const originalFetch = window.fetch.bind(window);

    function cookie(name) {
        const prefix = encodeURIComponent(name) + '=';
        const item = document.cookie.split(';').map(v => v.trim()).find(v => v.startsWith(prefix));
        return item ? decodeURIComponent(item.slice(prefix.length)) : '';
    }

    window.fetch = function securedFetch(input, init) {
        const options = Object.assign({}, init || {});
        const request = input instanceof Request ? input : null;
        const method = String(options.method || (request && request.method) || 'GET').toUpperCase();
        const target = new URL(request ? request.url : String(input), window.location.href);

        if (target.origin === window.location.origin && !['GET', 'HEAD', 'OPTIONS'].includes(method)) {
            const headers = new Headers(options.headers || (request && request.headers) || undefined);
            const token = cookie('csrf_token');
            if (token) headers.set('X-CSRF-Token', token);
            options.headers = headers;
        }
        return originalFetch(input, options);
    };
})();
