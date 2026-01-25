<script lang="ts">
	import { pb } from './lib/pb';
	import { AuthStore } from '@/lib/stores.svelte';
	import { Home, Login } from '@/pages';

	const authStore = new AuthStore();
	$effect(() => {
		if (pb.authStore.isValid) {
			pb.collection('_superusers')
				.authRefresh()
				.catch(() => {
					pb.authStore.clear();
				});
		}
	});

	let path = $state(window.location.pathname);
	$effect(() => {
		const onPopState = () => {
			path = window.location.pathname;
		};
		window.addEventListener('popstate', onPopState);

		return () => {
			window.removeEventListener('popstate', onPopState);
		};
	});
</script>

<h1 class="text-3xl font-semibold">Tado API Proxy</h1>

{#if !authStore.isValid}
	<Login />
{:else if path === '/'}
	<Home />
{/if}
