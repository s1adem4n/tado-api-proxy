<script module lang="ts">
	import type { Snippet } from 'svelte';

	export const snippets: Record<string, Snippet | null> = $state({});
</script>

<script>
	let { name }: { name: string } = $props();
</script>

{@render snippets[name]?.()}
