import { UserManager, WebStorageStateStore, type User } from 'oidc-client-ts';

// ── Configuration ───────────────────────────────────────────────────────────

const authority = import.meta.env.VITE_OIDC_AUTHORITY ?? 'https://auth.spencerbaumruk.com/realms/master';
const clientId = import.meta.env.VITE_OIDC_CLIENT_ID ?? 'scrabble';
const redirectUri = `${window.location.origin}/auth/callback`;
const postLogoutUri = window.location.origin;

const userManager = new UserManager({
	authority,
	client_id: clientId,
	redirect_uri: redirectUri,
	post_logout_redirect_uri: postLogoutUri,
	response_type: 'code',
	scope: 'openid profile email',
	automaticSilentRenew: true,
	userStore: new WebStorageStateStore({ store: localStorage })
});

// ── Reactive auth state ─────────────────────────────────────────────────────

type AuthListener = (user: User | null) => void;
let listeners: AuthListener[] = [];
let currentUser: User | null = null;

function notify(user: User | null) {
	currentUser = user;
	for (const fn of listeners) fn(user);
}

export function onAuthChange(fn: AuthListener): () => void {
	listeners.push(fn);
	// Immediately call with current state
	fn(currentUser);
	return () => {
		listeners = listeners.filter((l) => l !== fn);
	};
}

// ── Public API ──────────────────────────────────────────────────────────────

/** Redirect to Keycloak login page. */
export function login(): Promise<void> {
	return userManager.signinRedirect();
}

/** Redirect to Keycloak logout. */
export function logout(): Promise<void> {
	return userManager.signoutRedirect();
}

/** Handle the OIDC callback redirect. Returns the logged-in user. */
export async function handleCallback(): Promise<User> {
	const user = await userManager.signinRedirectCallback();
	notify(user);
	return user;
}

/** Get the current access token, or null if not logged in. */
export async function getAccessToken(): Promise<string | null> {
	const user = await userManager.getUser();
	if (!user || user.expired) return null;
	return user.access_token;
}

/** Get the current user, or null if not logged in. */
export async function getUser(): Promise<User | null> {
	const user = await userManager.getUser();
	if (!user || user.expired) return null;
	return user;
}

/** Initialize auth state on app load. */
export async function initAuth(): Promise<User | null> {
	const user = await userManager.getUser();
	if (user && !user.expired) {
		notify(user);
		return user;
	}
	notify(null);
	return null;
}

// Listen for token renewal and sign-out events
userManager.events.addUserLoaded((user) => notify(user));
userManager.events.addUserUnloaded(() => notify(null));
userManager.events.addSilentRenewError(() => notify(null));
