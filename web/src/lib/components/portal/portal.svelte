<script lang="ts">
	import { snippets } from './portal-output.svelte';
	import type { Snippet } from 'svelte';

	const {
		children,
		name
	}: {
		children: Snippet;
		name: string;
	} = $props();

	// svelte-ignore state_referenced_locally
	snippets[name] = children;

	$effect(() => {
		snippets[name] = children;
		return () => {
			snippets[name] = null;
		};
	});
</script>
