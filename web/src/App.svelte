<script lang="ts">
	import { pb } from './lib/pb';
	import { AuthStore, navigation } from '@/lib/stores.svelte';
	import { Home, Login, Statistics } from '@/pages';

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
</script>

{#if !authStore.isValid}
	<h1 class="text-3xl font-semibold">Tado API Proxy</h1>
	<Login />
{:else if navigation.path === '/'}
	<Home />
{:else if navigation.path === '/statistics'}
	<Statistics />
{:else}
	<p class="text-base-content/70">Page not found.</p>
{/if}
