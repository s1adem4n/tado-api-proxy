<script lang="ts">
	import { AccountsTable } from '@/lib/components/accounts-table';
	import { DeviceCodeSection } from '@/lib/components/device-code';
	import { TokensTable } from '@/lib/components/tokens-table';
	import { pb } from '@/lib/pb';
	import { MultipleSubscription } from '@/lib/stores.svelte';

	const accounts = new MultipleSubscription(pb.collection('accounts'));
	const homes = new MultipleSubscription(pb.collection('homes'));
	const tokens = new MultipleSubscription(pb.collection('tokens'));
	const clients = new MultipleSubscription(pb.collection('clients'));
	const codes = new MultipleSubscription(pb.collection('codes'));
</script>

<AccountsTable accounts={accounts.items} homes={homes.items} />

<DeviceCodeSection clients={clients.items} codes={codes.items} />

<TokensTable tokens={tokens.items} clients={clients.items} accounts={accounts.items} />
