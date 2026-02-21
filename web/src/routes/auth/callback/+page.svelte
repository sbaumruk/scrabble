<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { handleCallback } from '$lib/auth';

	let error = $state('');

	onMount(async () => {
		try {
			await handleCallback();
			goto('/');
		} catch (e) {
			console.error('Auth callback failed:', e);
			error = (e as Error).message;
			// Redirect home after a delay even on error
			setTimeout(() => goto('/'), 3000);
		}
	});
</script>

{#if error}
	<p class="error">Authentication error: {error}</p>
	<p class="redirect">Redirecting home...</p>
{:else}
	<p class="loading">Signing in...</p>
{/if}

<style>
	.loading, .redirect {
		text-align: center;
		padding: 60px;
		color: var(--text-muted);
	}
	.error {
		text-align: center;
		padding: 40px;
		color: #e53e3e;
	}
</style>
