<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';

	let { children } = $props();

	let theme = $state('light');

	onMount(() => {
		const saved = localStorage.getItem('theme');
		if (saved) {
			theme = saved;
		} else if (window.matchMedia('(prefers-color-scheme: dark)').matches) {
			theme = 'dark';
		}
		document.documentElement.setAttribute('data-theme', theme);
	});

	function toggleTheme() {
		theme = theme === 'light' ? 'dark' : 'light';
		localStorage.setItem('theme', theme);
		document.documentElement.setAttribute('data-theme', theme);
	}
</script>

<div class="app">
	<header>
		<a href="/" class="logo">Scrabble</a>
		<button class="theme-toggle" onclick={toggleTheme} title="Toggle theme">
			{theme === 'light' ? '\u263C' : '\u263E'}
		</button>
	</header>
	<main>
		{@render children()}
	</main>
</div>

<style>
	.app {
		min-height: 100vh;
		display: flex;
		flex-direction: column;
	}

	header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 12px 16px;
		border-bottom: 1px solid var(--border);
		background: var(--page-bg);
		position: sticky;
		top: 0;
		z-index: 10;
	}

	.logo {
		font-size: 18px;
		font-weight: 700;
		text-decoration: none;
		color: var(--text-primary);
		letter-spacing: -0.02em;
	}

	.theme-toggle {
		background: none;
		border: 1px solid var(--border);
		border-radius: 6px;
		padding: 4px 10px;
		font-size: 18px;
		color: var(--text-primary);
		transition: background 0.1s;
	}

	.theme-toggle:hover {
		background: var(--surface-hover);
	}

	main {
		flex: 1;
		padding: 16px;
		max-width: 1100px;
		width: 100%;
		margin: 0 auto;
	}
</style>
