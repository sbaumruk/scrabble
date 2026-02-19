<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { listBoards, createBoard } from '$lib/api';

	let boards = $state<string[]>([]);
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
			await createBoard(name);
			newName = '';
			boards = await listBoards();
		} catch (e) {
			alert('Failed to create board: ' + (e as Error).message);
		}
		creating = false;
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') handleCreate();
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
			{#each boards as name}
				<button class="board-card" onclick={() => goto(`/game?board=${encodeURIComponent(name)}`)}>
					<span class="board-name">{name}</span>
					<span class="arrow">&rarr;</span>
				</button>
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
		justify-content: space-between;
		padding: 14px 16px;
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: 8px;
		text-align: left;
		font-size: 15px;
		transition: background 0.1s;
	}

	.board-card:hover {
		background: var(--surface-hover);
	}

	.board-name {
		font-weight: 500;
		color: var(--text-primary);
	}

	.arrow {
		color: var(--text-muted);
		font-size: 18px;
	}
</style>
