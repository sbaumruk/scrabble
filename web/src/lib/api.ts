import type { Move, Ruleset, BoardMeta, BoardRecord } from './types';
import { getAccessToken } from './auth';

// In dev mode, Vite serves on :5173 but the Go API is on :8080.
// In production, both are served from the same origin.
const API_BASE = import.meta.env.DEV ? 'http://localhost:8080' : '';

async function fetchJSON<T>(url: string, opts?: RequestInit): Promise<T> {
	const headers: Record<string, string> = {
		...((opts?.headers as Record<string, string>) ?? {})
	};

	// Attach auth token if available
	const token = await getAccessToken();
	if (token) {
		headers['Authorization'] = `Bearer ${token}`;
	}

	const res = await fetch(API_BASE + url, { ...opts, headers });
	if (!res.ok) {
		const body = await res.json().catch(() => ({ error: res.statusText }));
		throw new Error(body.error || res.statusText);
	}
	return res.json();
}

// ── Board CRUD ──────────────────────────────────────────────────────────────

export async function listBoards(): Promise<BoardMeta[]> {
	const data = await fetchJSON<{ boards: BoardMeta[] }>('/api/boards');
	return data.boards;
}

export async function getBoard(id: string): Promise<BoardRecord> {
	return fetchJSON(`/api/boards/${encodeURIComponent(id)}`);
}

export async function saveBoard(id: string, board: string[]): Promise<void> {
	await fetchJSON(`/api/boards/${encodeURIComponent(id)}`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ board })
	});
}

export async function createBoard(name: string): Promise<{ ok: boolean; id: string }> {
	return fetchJSON('/api/boards', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ name })
	});
}

export async function deleteBoard(id: string): Promise<void> {
	await fetchJSON(`/api/boards/${encodeURIComponent(id)}`, {
		method: 'DELETE'
	});
}

// ── Sharing ─────────────────────────────────────────────────────────────────

export async function shareBoard(id: string): Promise<string> {
	const data = await fetchJSON<{ shareToken: string }>(
		`/api/boards/${encodeURIComponent(id)}/share`,
		{ method: 'POST' }
	);
	return data.shareToken;
}

export async function getSharedBoard(token: string): Promise<{ id: string; name: string; board: string[] }> {
	return fetchJSON(`/api/boards/shared/${encodeURIComponent(token)}`);
}

// ── Solver (stateless) ──────────────────────────────────────────────────────

export async function solve(board: string[], rack: string): Promise<Move[]> {
	const data = await fetchJSON<{ moves: Move[] }>('/api/solve', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ board, rack })
	});
	return data.moves;
}

export async function findOpponentPlacements(board: string[], word: string): Promise<Move[]> {
	const data = await fetchJSON<{ placements: Move[] }>('/api/opponent', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ board, word })
	});
	return data.placements;
}

export async function getRuleset(): Promise<Ruleset> {
	return fetchJSON('/api/ruleset');
}

// ── Auth ─────────────────────────────────────────────────────────────────────

export async function getMe(): Promise<{ sub: string; email: string; preferred_username: string }> {
	return fetchJSON('/api/me');
}
