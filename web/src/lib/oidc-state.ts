const OIDC_STATE_COOKIE = "oidc_state";
const OIDC_STATE_MAX_AGE = 600;

export function createOidcState(): string {
  const bytes = new Uint8Array(32);
  crypto.getRandomValues(bytes);
  const state = Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
  const secure = window.location.protocol === "https:" ? "; Secure" : "";
  document.cookie = `${OIDC_STATE_COOKIE}=${encodeURIComponent(state)}; Path=/; Max-Age=${OIDC_STATE_MAX_AGE}; SameSite=Lax${secure}`;
  return state;
}
