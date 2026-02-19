<script lang="ts">
	import type { Move } from '$lib/types';

	let {
		moves,
		selectedIndex = -1,
		onselect,
		onconfirm
	}: {
		moves: Move[];
		selectedIndex?: number;
		onselect: (index: number) => void;
		onconfirm: () => void;
	} = $props();

	function dirLabel(dir: string): string {
		return dir === 'H' ? 'across' : 'down';
	}
</script>

{#if moves.length > 0}
	<div class="move-list">
		{#each moves as move, i}
			<button
				class="move-item"
				class:selected={i === selectedIndex}
				onclick={() => onselect(i)}
				ondblclick={onconfirm}
			>
				<span class="rank">{i + 1}.</span>
				<span class="word">{move.word}</span>
				<span class="details">
					({move.x + 1},{move.y + 1}) {dirLabel(move.dir)}
				</span>
				<span class="score">{move.score}</span>
			</button>
		{/each}
	</div>
{:else}
	<div class="empty">No moves found</div>
{/if}

<style>
	.move-list {
		display: flex;
		flex-direction: column;
		gap: 2px;
		max-height: 300px;
		overflow-y: auto;
	}

	.move-item {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 8px 12px;
		background: var(--surface);
		color: var(--text-primary);
		border: 1px solid transparent;
		border-radius: 6px;
		text-align: left;
		transition: background 0.1s;
		font-size: 14px;
	}

	.move-item:hover {
		background: var(--surface-hover);
	}

	.move-item.selected {
		background: var(--surface-hover);
		border-color: var(--accent);
	}

	.rank {
		color: var(--text-muted);
		min-width: 24px;
		font-variant-numeric: tabular-nums;
	}

	.word {
		font-weight: 600;
		flex: 1;
	}

	.details {
		color: var(--text-secondary);
		font-size: 12px;
	}

	.score {
		font-weight: 700;
		color: var(--accent);
		min-width: 36px;
		text-align: right;
		font-variant-numeric: tabular-nums;
	}

	.empty {
		padding: 20px;
		text-align: center;
		color: var(--text-muted);
	}
</style>
