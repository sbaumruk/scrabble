<script lang="ts">
	let {
		value = $bindable(''),
		onsubmit
	}: {
		value?: string;
		onsubmit: (rack: string) => void;
	} = $props();

	let inputEl: HTMLInputElement;

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter' && value.trim()) {
			onsubmit(value.trim().toUpperCase());
		}
	}

	function handleInput(e: Event) {
		const target = e.target as HTMLInputElement;
		// Allow only letters and asterisk, max 7
		value = target.value.replace(/[^a-zA-Z*]/g, '').slice(0, 7).toUpperCase();
	}
</script>

<div class="rack-input">
	<div class="tiles">
		{#each { length: 7 } as _, i}
			<div class="tile-slot" class:filled={i < value.length}>
				{#if i < value.length}
					<span class="tile-letter">{value[i]}</span>
				{/if}
			</div>
		{/each}
	</div>
	<input
		bind:this={inputEl}
		bind:value={value}
		oninput={handleInput}
		onkeydown={handleKeydown}
		placeholder="Enter tiles (* = blank)"
		maxlength="7"
		autocomplete="off"
		autocapitalize="characters"
		spellcheck="false"
	/>
</div>

<style>
	.rack-input {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.tiles {
		display: flex;
		gap: 4px;
		justify-content: center;
	}

	.tile-slot {
		width: 36px;
		height: 36px;
		background: var(--surface);
		border: 2px solid var(--border);
		border-radius: 4px;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.tile-slot.filled {
		background: var(--cell-tile);
		border-color: var(--cell-tile-border);
	}

	.tile-letter {
		font-weight: 700;
		font-size: 18px;
		color: var(--text-on-tile);
	}

	input {
		padding: 8px 12px;
		border: 1px solid var(--border);
		border-radius: 6px;
		background: var(--surface);
		color: var(--text-primary);
		font-size: 16px;
		text-align: center;
		text-transform: uppercase;
		letter-spacing: 0.15em;
	}

	input::placeholder {
		text-transform: none;
		letter-spacing: normal;
		color: var(--text-muted);
	}

	input:focus {
		outline: 2px solid var(--accent);
		outline-offset: -1px;
	}
</style>
