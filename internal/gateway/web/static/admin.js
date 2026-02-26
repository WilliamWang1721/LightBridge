(() => {
  'use strict';

  const TOAST_STYLE_ID = 'lb-toast-style';
  const TOAST_HOST_ID = 'lb-toast-host';

  function ensureToastStyles() {
    if (document.getElementById(TOAST_STYLE_ID)) return;
    const style = document.createElement('style');
    style.id = TOAST_STYLE_ID;
    style.textContent = `
      .lb-toast-host{
        position:fixed;top:16px;right:16px;z-index:9999;
        display:flex;flex-direction:column;gap:10px;
      }
      .lb-toast{
        max-width:360px;padding:10px 12px;border:1px solid rgba(0,0,0,.12);
        background:#fff;color:#111;
        box-shadow:0 12px 30px rgba(0,0,0,.12);
        font: 13px/1.45 ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, Helvetica, Arial;
        white-space:pre-wrap;word-break:break-word;
      }
      .lb-toast--ok{border-color:#bbf7d0;background:#f0fdf4;color:#166534;}
      .lb-toast--error{border-color:#fed7d7;background:#fff5f5;color:#b42318;}
      .lb-toast--info{border-color:#e5e7eb;background:#fafafa;color:#111827;}
    `;
    document.head.appendChild(style);
  }

  function ensureToastHost() {
    let host = document.getElementById(TOAST_HOST_ID);
    if (host) return host;
    ensureToastStyles();
    host = document.createElement('div');
    host.id = TOAST_HOST_ID;
    host.className = 'lb-toast-host';
    document.body.appendChild(host);
    return host;
  }

  function toast(message, variant = 'info', timeoutMs = 2600) {
    try {
      const host = ensureToastHost();
      const node = document.createElement('div');
      node.className = `lb-toast lb-toast--${variant}`;
      node.textContent = String(message ?? '');
      host.appendChild(node);
      window.setTimeout(() => {
        node.remove();
        if (host.childElementCount === 0) host.remove();
      }, timeoutMs);
    } catch (_) {
      // Fallback: do nothing.
    }
  }

  function textFromResponseBody(body) {
    if (!body) return '';
    if (typeof body === 'string') return body;
    if (typeof body.error === 'string') return body.error;
    if (body.error && typeof body.error.message === 'string') return body.error.message;
    return '';
  }

  async function apiJSON(url, options = {}) {
    const opts = { ...options };
    const headers = { accept: 'application/json', ...(opts.headers || {}) };
    if (opts.body !== undefined && !(opts.body instanceof FormData) && typeof opts.body !== 'string') {
      headers['content-type'] = headers['content-type'] || 'application/json';
      opts.body = JSON.stringify(opts.body);
    }
    opts.headers = headers;
    const res = await fetch(url, opts);

    const raw = await res.text();
    let body = null;
    try {
      body = raw ? JSON.parse(raw) : null;
    } catch (_) {
      body = raw ? { raw } : null;
    }

    if (res.status === 401) {
      location.href = '/admin/login';
      throw new Error('login required');
    }
    if (res.status === 403 && textFromResponseBody(body).includes('setup')) {
      location.href = '/admin/setup';
      throw new Error('setup required');
    }
    if (!res.ok) {
      const msg = textFromResponseBody(body) || res.statusText || 'request failed';
      const err = new Error(msg);
      err.status = res.status;
      err.body = body;
      throw err;
    }
    return body;
  }

  function formatCompactNumber(n) {
    const num = Number(n);
    if (!Number.isFinite(num)) return '-';
    return new Intl.NumberFormat(undefined).format(num);
  }

  function formatTokens(tokens) {
    const n = Number(tokens);
    if (!Number.isFinite(n)) return '-';
    const abs = Math.abs(n);
    if (abs >= 1e9) return `${(n / 1e9).toFixed(1)}B`;
    if (abs >= 1e6) return `${(n / 1e6).toFixed(1)}M`;
    if (abs >= 1e3) return `${(n / 1e3).toFixed(1)}K`;
    return String(Math.round(n));
  }

  function formatISODateTime(iso) {
    if (!iso) return '';
    const d = new Date(iso);
    if (Number.isNaN(d.getTime())) return String(iso);
    return d.toLocaleString();
  }

  function el(tag, attrs = {}, children = []) {
    const node = document.createElement(tag);
    for (const [k, v] of Object.entries(attrs || {})) {
      if (v === undefined || v === null) continue;
      if (k === 'className') {
        node.className = String(v);
      } else if (k === 'text') {
        node.textContent = String(v);
      } else if (k === 'dataset' && typeof v === 'object') {
        for (const [dk, dv] of Object.entries(v)) node.dataset[dk] = String(dv);
      } else if (k.startsWith('on') && typeof v === 'function') {
        node.addEventListener(k.slice(2).toLowerCase(), v);
      } else if (k in node) {
        node[k] = v;
      } else {
        node.setAttribute(k, String(v));
      }
    }
    for (const child of Array.isArray(children) ? children : [children]) {
      if (child === undefined || child === null) continue;
      if (typeof child === 'string') node.appendChild(document.createTextNode(child));
      else node.appendChild(child);
    }
    return node;
  }

  window.LightBridgeAdmin = {
    apiJSON,
    toast,
    el,
    formatCompactNumber,
    formatTokens,
    formatISODateTime,
  };
})();

