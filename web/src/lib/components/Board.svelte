<script lang="ts">
	import type { Move, Ruleset } from '$lib/types';

	let {
		board,
		ruleset,
		preview = null
	}: {
		board: string[];
		ruleset: Ruleset | null;
		preview?: Move | null;
	} = $props();

	function cti(x: number, y: number): string {
		return `${x},${y}`;
	}

	// Build lookup sets from ruleset
	let twSet = $derived(new Set((ruleset?.tripleWord ?? []).map(([x, y]) => cti(x, y))));
	let dwSet = $derived(new Set((ruleset?.doubleWord ?? []).map(([x, y]) => cti(x, y))));
	let tlSet = $derived(new Set((ruleset?.tripleLetter ?? []).map(([x, y]) => cti(x, y))));
	let dlSet = $derived(new Set((ruleset?.doubleLetter ?? []).map(([x, y]) => cti(x, y))));

	// Build preview overlay
	let previewMap = $derived.by(() => {
		const map = new Map<string, string>();
		if (!preview || !board) return map;
		let tileIdx = 0;
		const tiles = preview.tiles;
		if (preview.dir === 'V') {
			for (let i = preview.y; tileIdx < tiles.length && i < 15; i++) {
				if (board[i]?.[preview.x] && board[i][preview.x] !== '.') continue;
				map.set(cti(preview.x, i), tiles[tileIdx]);
				tileIdx++;
			}
		} else {
			for (let i = preview.x; tileIdx < tiles.length && i < 15; i++) {
				if (board[preview.y]?.[i] && board[preview.y][i] !== '.') continue;
				map.set(cti(i, preview.y), tiles[tileIdx]);
				tileIdx++;
			}
		}
		return map;
	});

	let letterPoints = $derived(ruleset?.letterPoints ?? {});

	function getPointValue(letter: string): number {
		// Lowercase = blank tile, scores 0
		if (letter >= 'a' && letter <= 'z') return 0;
		return letterPoints[letter.toUpperCase()] ?? 0;
	}

	function cellType(x: number, y: number): string {
		const key = cti(x, y);
		const boardChar = board[y]?.[x] ?? '.';
		const previewChar = previewMap.get(key);

		if (previewChar) return 'preview';
		if (boardChar !== '.') return 'tile';
		if (x === 7 && y === 7) return 'center';
		if (twSet.has(key)) return 'tw';
		if (dwSet.has(key)) return 'dw';
		if (tlSet.has(key)) return 'tl';
		if (dlSet.has(key)) return 'dl';
		return 'empty';
	}

	function cellLetter(x: number, y: number): string {
		const key = cti(x, y);
		const previewChar = previewMap.get(key);
		if (previewChar) return previewChar.toUpperCase();
		const boardChar = board[y]?.[x] ?? '.';
		if (boardChar !== '.') return boardChar.toUpperCase();
		return '';
	}

	function cellLabel(x: number, y: number): string {
		const key = cti(x, y);
		if (x === 7 && y === 7) return '\u2605';
		if (twSet.has(key)) return 'TW';
		if (dwSet.has(key)) return 'DW';
		if (tlSet.has(key)) return 'TL';
		if (dlSet.has(key)) return 'DL';
		return '';
	}

	function cellPointValue(x: number, y: number): number {
		const letter = cellLetter(x, y);
		if (!letter) return 0;
		const key = cti(x, y);
		const previewChar = previewMap.get(key);
		const boardChar = board[y]?.[x] ?? '.';
		// Use the raw char (before toUpperCase) to detect blanks
		const rawChar = previewChar ?? boardChar;
		return getPointValue(rawChar);
	}
</script>

<div class="board">
	{#each { length: 15 } as _, y}
		{#each { length: 15 } as _, x}
			{@const type = cellType(x, y)}
			{@const letter = cellLetter(x, y)}
			{@const label = cellLabel(x, y)}
			{@const pts = cellPointValue(x, y)}
			<div class="cell {type}">
				{#if letter}
					<span class="letter">{letter}</span>
					{#if pts > 0}
						<span class="points">{pts}</span>
					{/if}
				{:else if label}
					<span class="label">{label}</span>
				{/if}
			</div>
		{/each}
	{/each}
</div>

<style>
	.board {
		display: grid;
		grid-template-columns: repeat(15, 1fr);
		gap: 1px;
		aspect-ratio: 1;
		max-width: min(95vw, 600px);
		background: var(--board-gap);
		border-radius: 4px;
		overflow: hidden;
		margin: 0 auto;
	}

	.cell {
		aspect-ratio: 1;
		display: flex;
		align-items: center;
		justify-content: center;
		position: relative;
		font-size: clamp(8px, 2.5vw, 16px);
	}

	.cell.empty, .cell.center { background: var(--cell-empty); }
	.cell.tw { background: var(--cell-tw); }
	.cell.dw { background: var(--cell-dw); }
	.cell.tl { background: var(--cell-tl); }
	.cell.dl { background: var(--cell-dl); }

	.cell.tile {
		background: var(--cell-tile);
		border: 1px solid var(--cell-tile-border);
		border-radius: 2px;
	}

	.cell.preview {
		background: var(--cell-preview);
		border: 1px solid var(--cell-preview-border);
		border-radius: 2px;
	}

	.letter {
		font-weight: 700;
		color: var(--text-on-tile);
		line-height: 1;
	}

	.points {
		position: absolute;
		bottom: 1px;
		right: 2px;
		font-size: clamp(5px, 1.2vw, 8px);
		color: var(--text-secondary);
		font-weight: 500;
		line-height: 1;
	}

	.label {
		font-size: clamp(5px, 1.4vw, 9px);
		font-weight: 600;
		color: var(--text-multiplier);
		letter-spacing: 0.02em;
		line-height: 1;
	}
</style>
