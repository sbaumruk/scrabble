import type { Move, Ruleset } from './types';

// In dev mode, Vite serves on :5173 but the Go API is on :8080.
// In production, both are served from the same origin.
const API_BASE = import.meta.env.DEV ? 'http://localhost:8080' : '';

async function fetchJSON<T>(url: string, opts?: RequestInit): Promise<T> {
	const res = await fetch(API_BASE + url, opts);
	if (!res.ok) {
		const body = await res.json().catch(() => ({ error: res.statusText }));
		throw new Error(body.error || res.statusText);
	}
	return res.json();
}

export async function listBoards(): Promise<string[]> {
	const data = await fetchJSON<{ boards: string[] }>('/api/boards');
	return data.boards;
}

export async function getBoard(name: string): Promise<{ name: string; board: string[] }> {
	return fetchJSON(`/api/boards/${encodeURIComponent(name)}`);
}

export async function saveBoard(name: string, board: string[]): Promise<void> {
	await fetchJSON(`/api/boards/${encodeURIComponent(name)}`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ board })
	});
}

export async function createBoard(name: string): Promise<void> {
	await fetchJSON('/api/boards', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ name })
	});
}

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
