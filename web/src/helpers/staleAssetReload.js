const reloadKey = 'moleapi:stale-asset-reload-at';
const reloadCooldownMs = 5 * 60 * 1000;

export function isStaleAssetError(error) {
  const message = String(error?.message || error || '');
  return (
    message.includes('Failed to fetch dynamically imported module') ||
    message.includes('Importing a module script failed') ||
    message.includes('error loading dynamically imported module') ||
    message.includes('Unable to preload CSS')
  );
}

export function reloadForStaleAssets(error, options = {}) {
  if (typeof window === 'undefined') {
    return false;
  }

  if (!options.force && !isStaleAssetError(error)) {
    return false;
  }

  try {
    const lastReloadAt = Number(sessionStorage.getItem(reloadKey) || 0);
    const now = Date.now();
    if (now - lastReloadAt < reloadCooldownMs) {
      return false;
    }
    sessionStorage.setItem(reloadKey, String(now));
  } catch {
    // Storage may be unavailable in strict privacy modes; a single reload is still useful.
  }

  window.location.reload();
  return true;
}

let staleAssetReloadRegistered = false;

export function registerStaleAssetReload() {
  if (typeof window === 'undefined' || staleAssetReloadRegistered) {
    return;
  }

  staleAssetReloadRegistered = true;

  window.addEventListener('vite:preloadError', (event) => {
    event.preventDefault();
    reloadForStaleAssets(event.payload, { force: true });
  });

  window.addEventListener('unhandledrejection', (event) => {
    reloadForStaleAssets(event.reason);
  });
}
