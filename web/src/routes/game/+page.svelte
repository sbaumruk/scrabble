<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import Board from '$lib/components/Board.svelte';
	import MoveList from '$lib/components/MoveList.svelte';
	import RackInput from '$lib/components/RackInput.svelte';
	import {
		getBoard,
		saveBoard,
		solve,
		findOpponentPlacements,
		getRuleset,
		getSharedBoard,
		shareBoard
	} from '$lib/api';
	import type { Move, Ruleset } from '$lib/types';

	let boardId = $state('');
	let boardName = $state('');
	let board = $state<string[]>([]);
	let ruleset = $state<Ruleset | null>(null);
	let loading = $state(true);
	let isReadOnly = $state(false);
	let shareToken = $state<string | null>(null);

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

	// Share UI
	let shareUrl = $state('');
	let showShareCopied = $state(false);

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
		const id = params.get('id');
		const shared = params.get('shared');

		if (!id && !shared) {
			goto('/');
			return;
		}

		try {
			const rulesetData = getRuleset();

			if (shared) {
				// Shared board (read-only)
				const sharedData = await getSharedBoard(shared);
				boardId = sharedData.id;
				boardName = sharedData.name;
				board = sharedData.board;
				isReadOnly = true;
				shareToken = shared;
			} else if (id) {
				const boardData = await getBoard(id);
				boardId = boardData.id;
				boardName = boardData.name;
				board = boardData.board;
				// Read-only if user doesn't own this board
				isReadOnly = boardData.isOwner === false;
			}

			ruleset = await rulesetData;
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
		if (!isReadOnly && boardId) {
			try {
				await saveBoard(boardId, board);
			} catch (e) {
				console.error('Failed to save:', e);
			}
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
		if (!isReadOnly && boardId) {
			try {
				await saveBoard(boardId, board);
			} catch (e) {
				console.error('Failed to save:', e);
			}
		}
		statusMsg = `Opponent played ${move.word} for ${move.score} points`;
		opponentMoves = [];
		oppSelectedIndex = -1;
		opponentWord = '';
		phase = 'my-turn';
	}

	async function handleShare() {
		if (!boardId) return;
		try {
			const token = await shareBoard(boardId);
			shareUrl = `${window.location.origin}/game?shared=${token}`;
			await navigator.clipboard.writeText(shareUrl);
			showShareCopied = true;
			setTimeout(() => { showShareCopied = false; }, 2000);
		} catch (e) {
			alert('Failed to create share link: ' + (e as Error).message);
		}
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
			<div class="board-header">
				<div class="board-title">{boardName}</div>
				<div class="board-actions">
					{#if isReadOnly}
						<span class="badge read-only">Read-only</span>
					{:else if boardId}
						<button class="share-btn" onclick={handleShare} title="Share board">
							{showShareCopied ? 'Copied!' : 'Share'}
						</button>
					{/if}
				</div>
			</div>

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

	.board-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 8px;
	}

	.board-title {
		font-size: 16px;
		font-weight: 600;
		color: var(--text-secondary);
	}

	.board-actions {
		display: flex;
		gap: 6px;
		align-items: center;
	}

	.badge {
		font-size: 11px;
		padding: 3px 8px;
		border-radius: 4px;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.04em;
	}

	.badge.read-only {
		background: var(--surface);
		color: var(--text-muted);
		border: 1px solid var(--border);
	}

	.share-btn {
		padding: 4px 12px;
		font-size: 13px;
		font-weight: 500;
		border: 1px solid var(--border);
		border-radius: 6px;
		background: var(--surface);
		color: var(--text-secondary);
		cursor: pointer;
		transition: all 0.1s;
	}

	.share-btn:hover {
		background: var(--surface-hover);
		color: var(--text-primary);
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
