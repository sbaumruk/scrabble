<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import Board from '$lib/components/Board.svelte';
	import MoveList from '$lib/components/MoveList.svelte';
	import RackInput from '$lib/components/RackInput.svelte';
	import {
		getBoard,
		saveBoard,
		solve,
		findOpponentPlacements,
		getRuleset
	} from '$lib/api';
	import type { Move, Ruleset } from '$lib/types';

	let boardName = $state('');
	let board = $state<string[]>([]);
	let ruleset = $state<Ruleset | null>(null);
	let loading = $state(true);

	// My turn state
	let rack = $state('');
	let myMoves = $state<Move[]>([]);
	let mySelectedIndex = $state(-1);
	let solving = $state(false);

	// Opponent turn state
	let opponentWord = $state('');
	let opponentMoves = $state<Move[]>([]);
	let oppSelectedIndex = $state(-1);
	let findingPlacements = $state(false);

	// Which phase
	let phase = $state<'my-turn' | 'opponent-turn'>('my-turn');

	// Status messages
	let statusMsg = $state('');

	let previewMove = $derived.by(() => {
		if (phase === 'my-turn' && mySelectedIndex >= 0 && myMoves[mySelectedIndex]) {
			return myMoves[mySelectedIndex];
		}
		if (phase === 'opponent-turn' && oppSelectedIndex >= 0 && opponentMoves[oppSelectedIndex]) {
			return opponentMoves[oppSelectedIndex];
		}
		return null;
	});

	onMount(async () => {
		const params = new URLSearchParams(window.location.search);
		const name = params.get('board');
		if (!name) {
			goto('/');
			return;
		}
		boardName = name;
		try {
			const [boardData, rulesetData] = await Promise.all([
				getBoard(name),
				getRuleset()
			]);
			board = boardData.board;
			ruleset = rulesetData;
		} catch (e) {
			alert('Failed to load board: ' + (e as Error).message);
			goto('/');
			return;
		}
		loading = false;
	});

	function applyMoveToBoard(b: string[], move: Move): string[] {
		const rows = b.map((r) => [...r]);
		let tileIdx = 0;
		if (move.dir === 'V') {
			for (let i = move.y; tileIdx < move.tiles.length && i < 15; i++) {
				if (rows[i][move.x] !== '.') continue;
				rows[i][move.x] = move.tiles[tileIdx];
				tileIdx++;
			}
		} else {
			for (let i = move.x; tileIdx < move.tiles.length && i < 15; i++) {
				if (rows[move.y][i] !== '.') continue;
				rows[move.y][i] = move.tiles[tileIdx];
				tileIdx++;
			}
		}
		return rows.map((r) => r.join(''));
	}

	async function handleSolve(rackStr: string) {
		rack = rackStr;
		solving = true;
		statusMsg = '';
		myMoves = [];
		mySelectedIndex = -1;
		try {
			myMoves = await solve(board, rackStr);
			if (myMoves.length > 0) {
				mySelectedIndex = 0;
			} else {
				statusMsg = 'No valid moves found.';
			}
		} catch (e) {
			statusMsg = 'Error: ' + (e as Error).message;
		}
		solving = false;
	}

	async function handlePlayMyMove() {
		if (mySelectedIndex < 0 || !myMoves[mySelectedIndex]) return;
		const move = myMoves[mySelectedIndex];
		board = applyMoveToBoard(board, move);
		try {
			await saveBoard(boardName, board);
		} catch (e) {
			console.error('Failed to save:', e);
		}
		statusMsg = `Played ${move.word} for ${move.score} points`;
		myMoves = [];
		mySelectedIndex = -1;
		rack = '';
		phase = 'opponent-turn';
	}

	async function handleFindPlacements() {
		const word = opponentWord.trim().toUpperCase();
		if (!word) return;
		findingPlacements = true;
		statusMsg = '';
		opponentMoves = [];
		oppSelectedIndex = -1;
		try {
			opponentMoves = await findOpponentPlacements(board, word);
			if (opponentMoves.length > 0) {
				oppSelectedIndex = 0;
			} else {
				statusMsg = `No valid placement found for "${word}".`;
			}
		} catch (e) {
			statusMsg = 'Error: ' + (e as Error).message;
		}
		findingPlacements = false;
	}

	async function handleConfirmOpponent() {
		if (oppSelectedIndex < 0 || !opponentMoves[oppSelectedIndex]) return;
		const move = opponentMoves[oppSelectedIndex];
		board = applyMoveToBoard(board, move);
		try {
			await saveBoard(boardName, board);
		} catch (e) {
			console.error('Failed to save:', e);
		}
		statusMsg = `Opponent played ${move.word} for ${move.score} points`;
		opponentMoves = [];
		oppSelectedIndex = -1;
		opponentWord = '';
		phase = 'my-turn';
	}

	function handleOpponentKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') handleFindPlacements();
	}
</script>

{#if loading}
	<p class="loading">Loading...</p>
{:else}
	<div class="game">
		<div class="sidebar">
			<div class="board-title">{boardName}</div>

			{#if statusMsg}
				<div class="status">{statusMsg}</div>
			{/if}

			<!-- Phase tabs -->
			<div class="tabs">
				<button
					class="tab"
					class:active={phase === 'my-turn'}
					onclick={() => { phase = 'my-turn'; }}
				>My Turn</button>
				<button
					class="tab"
					class:active={phase === 'opponent-turn'}
					onclick={() => { phase = 'opponent-turn'; }}
				>Opponent</button>
			</div>

			{#if phase === 'my-turn'}
				<div class="section">
					<RackInput bind:value={rack} onsubmit={handleSolve} />
					<button
						class="btn primary"
						onclick={() => handleSolve(rack)}
						disabled={solving || !rack.trim()}
					>
						{solving ? 'Searching...' : 'Find Moves'}
					</button>
				</div>

				{#if myMoves.length > 0}
					<MoveList
						moves={myMoves}
						selectedIndex={mySelectedIndex}
						onselect={(i) => { mySelectedIndex = i; }}
						onconfirm={handlePlayMyMove}
					/>
					<button class="btn primary" onclick={handlePlayMyMove} disabled={mySelectedIndex < 0}>
						Play
					</button>
				{/if}
			{:else}
				<div class="section">
					<div class="opp-input">
						<input
							bind:value={opponentWord}
							onkeydown={handleOpponentKeydown}
							placeholder="Opponent's word"
							autocomplete="off"
							autocapitalize="characters"
							spellcheck="false"
						/>
						<button
							class="btn primary"
							onclick={handleFindPlacements}
							disabled={findingPlacements || !opponentWord.trim()}
						>
							{findingPlacements ? 'Searching...' : 'Find Placements'}
						</button>
					</div>
				</div>

				{#if opponentMoves.length > 0}
					<MoveList
						moves={opponentMoves}
						selectedIndex={oppSelectedIndex}
						onselect={(i) => { oppSelectedIndex = i; }}
						onconfirm={handleConfirmOpponent}
					/>
					<button class="btn primary" onclick={handleConfirmOpponent} disabled={oppSelectedIndex < 0}>
						Confirm
					</button>
				{/if}
			{/if}
		</div>

		<div class="board-area">
			<Board {board} {ruleset} preview={previewMove} />
		</div>
	</div>
{/if}

<style>
	.loading {
		text-align: center;
		padding: 60px;
		color: var(--text-muted);
	}

	.game {
		display: flex;
		flex-direction: column;
		gap: 16px;
	}

	.sidebar {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.board-title {
		font-size: 16px;
		font-weight: 600;
		color: var(--text-secondary);
	}

	.status {
		padding: 8px 12px;
		background: var(--surface);
		border-radius: 6px;
		font-size: 13px;
		color: var(--text-secondary);
	}

	.tabs {
		display: flex;
		gap: 4px;
		background: var(--surface);
		border-radius: 8px;
		padding: 3px;
	}

	.tab {
		flex: 1;
		padding: 8px;
		border: none;
		border-radius: 6px;
		background: transparent;
		font-size: 14px;
		font-weight: 500;
		color: var(--text-secondary);
		transition: all 0.1s;
	}

	.tab.active {
		background: var(--page-bg);
		color: var(--text-primary);
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
	}

	.section {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.opp-input {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.opp-input input {
		padding: 10px 12px;
		border: 1px solid var(--border);
		border-radius: 6px;
		background: var(--surface);
		color: var(--text-primary);
		font-size: 16px;
		text-transform: uppercase;
		letter-spacing: 0.1em;
	}

	.opp-input input:focus {
		outline: 2px solid var(--accent);
		outline-offset: -1px;
	}

	.btn {
		padding: 10px 20px;
		border: 1px solid var(--border);
		border-radius: 6px;
		font-size: 14px;
		font-weight: 600;
		background: var(--surface);
		color: var(--text-primary);
	}

	.btn.primary {
		background: var(--accent);
		color: var(--accent-text);
		border-color: transparent;
	}

	.btn.primary:hover:not(:disabled) {
		background: var(--accent-hover);
	}

	.btn:disabled {
		opacity: 0.5;
		cursor: default;
	}

	.board-area {
		display: flex;
		justify-content: center;
	}

	/* Desktop layout */
	@media (min-width: 768px) {
		.game {
			flex-direction: row;
		}

		.sidebar {
			width: 320px;
			flex-shrink: 0;
			order: -1;
		}

		.board-area {
			flex: 1;
			align-items: flex-start;
		}
	}
</style>
