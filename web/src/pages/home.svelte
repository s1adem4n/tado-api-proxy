<script lang="ts">
	import { AccountsTable } from '@/lib/components/accounts-table';
	import { DeviceCodeSection } from '@/lib/components/device-code';
	import { TokensTable } from '@/lib/components/tokens-table';
	import { pb } from '@/lib/pb';
	import { MultipleSubscription, navigation } from '@/lib/stores.svelte';
	import ChartBarIcon from '~icons/lucide/chart-bar';
	import LogOutIcon from '~icons/lucide/log-out';

	const accounts = new MultipleSubscription(pb.collection('accounts'));
	const homes = new MultipleSubscription(pb.collection('homes'));
	const tokens = new MultipleSubscription(pb.collection('tokens'));
	const clients = new MultipleSubscription(pb.collection('clients'));
	const codes = new MultipleSubscription(pb.collection('codes'));
</script>

<header class="flex items-center justify-between">
	<h1 class="text-3xl font-semibold">Tado API Proxy</h1>
	<div class="flex items-center gap-1">
		<button class="btn btn-ghost btn-sm" onclick={() => navigation.navigate('/statistics')}>
			<ChartBarIcon class="h-4 w-4" />
			Statistics
		</button>

		<button class="btn btn-ghost btn-sm" onclick={() => pb.authStore.clear()}>
			<LogOutIcon class="h-4 w-4" />
			Logout
		</button>
	</div>
</header>

<AccountsTable accounts={accounts.items} homes={homes.items} />

<DeviceCodeSection clients={clients.items} codes={codes.items} />

<TokensTable tokens={tokens.items} clients={clients.items} accounts={accounts.items} />
