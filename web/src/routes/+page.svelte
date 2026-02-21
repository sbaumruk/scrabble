<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { listBoards, createBoard, deleteBoard } from '$lib/api';
	import type { BoardMeta } from '$lib/types';

	let boards = $state<BoardMeta[]>([]);
	let newName = $state('');
	let loading = $state(true);
	let creating = $state(false);

	onMount(async () => {
		try {
			boards = await listBoards();
		} catch (e) {
			console.error('Failed to load boards:', e);
		}
		loading = false;
	});

	async function handleCreate() {
		const name = newName.trim();
		if (!name) return;
		creating = true;
		try {
			const result = await createBoard(name);
			newName = '';
			goto(`/game?id=${encodeURIComponent(result.id)}`);
		} catch (e) {
			alert('Failed to create board: ' + (e as Error).message);
			creating = false;
		}
	}

	async function handleDelete(board: BoardMeta) {
		if (!confirm(`Delete "${board.name}"? This cannot be undone.`)) return;
		try {
			await deleteBoard(board.id);
			boards = boards.filter((b) => b.id !== board.id);
		} catch (e) {
			alert('Failed to delete board: ' + (e as Error).message);
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') handleCreate();
	}

	function formatDate(dateStr: string): string {
		const d = new Date(dateStr);
		const now = new Date();
		const diffMs = now.getTime() - d.getTime();
		const diffMins = Math.floor(diffMs / 60000);
		if (diffMins < 1) return 'just now';
		if (diffMins < 60) return `${diffMins}m ago`;
		const diffHours = Math.floor(diffMins / 60);
		if (diffHours < 24) return `${diffHours}h ago`;
		const diffDays = Math.floor(diffHours / 24);
		if (diffDays < 7) return `${diffDays}d ago`;
		return d.toLocaleDateString();
	}
</script>

<div class="picker">
	<h1>Boards</h1>

	<div class="create-row">
		<input
			bind:value={newName}
			onkeydown={handleKeydown}
			placeholder="New board name"
			disabled={creating}
		/>
		<button onclick={handleCreate} disabled={creating || !newName.trim()}>
			{creating ? 'Creating...' : 'Create'}
		</button>
	</div>

	{#if loading}
		<p class="status">Loading...</p>
	{:else if boards.length === 0}
		<p class="status">No boards yet. Create one to get started.</p>
	{:else}
		<div class="board-list">
			{#each boards as board}
				<div class="board-card">
					<button class="board-main" onclick={() => goto(`/game?id=${encodeURIComponent(board.id)}`)}>
						<span class="board-name">{board.name}</span>
						<span class="board-date">{formatDate(board.updatedAt)}</span>
					</button>
					<button
						class="delete-btn"
						onclick={(e) => { e.stopPropagation(); handleDelete(board); }}
						title="Delete board"
					>
						&times;
					</button>
				</div>
			{/each}
		</div>
	{/if}
</div>

<style>
	.picker {
		max-width: 480px;
		margin: 0 auto;
	}

	h1 {
		font-size: 24px;
		font-weight: 700;
		margin-bottom: 16px;
	}

	.create-row {
		display: flex;
		gap: 8px;
		margin-bottom: 24px;
	}

	.create-row input {
		flex: 1;
		padding: 10px 12px;
		border: 1px solid var(--border);
		border-radius: 6px;
		background: var(--surface);
		color: var(--text-primary);
		font-size: 14px;
	}

	.create-row input:focus {
		outline: 2px solid var(--accent);
		outline-offset: -1px;
	}

	.create-row button {
		padding: 10px 20px;
		background: var(--accent);
		color: var(--accent-text);
		border: none;
		border-radius: 6px;
		font-weight: 600;
		font-size: 14px;
	}

	.create-row button:hover:not(:disabled) {
		background: var(--accent-hover);
	}

	.create-row button:disabled {
		opacity: 0.5;
		cursor: default;
	}

	.status {
		color: var(--text-muted);
		text-align: center;
		padding: 40px 0;
	}

	.board-list {
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.board-card {
		display: flex;
		align-items: center;
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: 8px;
		transition: background 0.1s;
	}

	.board-card:hover {
		background: var(--surface-hover);
	}

	.board-main {
		flex: 1;
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 14px 16px;
		background: none;
		border: none;
		text-align: left;
		font-size: 15px;
		cursor: pointer;
		color: inherit;
	}

	.board-name {
		font-weight: 500;
		color: var(--text-primary);
	}

	.board-date {
		color: var(--text-muted);
		font-size: 13px;
	}

	.delete-btn {
		padding: 8px 12px;
		background: none;
		border: none;
		color: var(--text-muted);
		font-size: 20px;
		line-height: 1;
		cursor: pointer;
		border-radius: 0 8px 8px 0;
		transition: color 0.1s;
	}

	.delete-btn:hover {
		color: #e53e3e;
	}
</style>
