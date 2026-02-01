<script lang="ts">
	import TokensTableRow from './tokens-table-row.svelte';
	import {
		fetchRatelimits,
		type Account,
		type Client,
		type Ratelimits,
		type Token
	} from '@/lib/pb';

	let {
		tokens,
		accounts,
		clients
	}: {
		tokens: Token[];
		accounts: Account[];
		clients: Client[];
	} = $props();

	const sortedTokens = $derived(
		tokens.toSorted((a, b) => {
			const accountCompare = a.account.localeCompare(b.account);
			if (accountCompare !== 0) return accountCompare;
			return a.client.localeCompare(b.client);
		})
	);

	let ratelimits: Ratelimits | null = $state({});
	$effect(() => {
		tokens;
		fetchRatelimits().then((data) => {
			ratelimits = data;
		});
	});
</script>

<div class="flex flex-col gap-2">
	<h2 class="text-2xl font-semibold">Tokens</h2>
	<p class="text-sm text-base-content/70">Active API tokens and their rate limit usage.</p>

	<div class="overflow-x-auto rounded-box border border-base-content/5 bg-base-100">
		<table class="table">
			<thead>
				<tr>
					<th>Account</th>
					<th>Client</th>
					<th>Last used</th>
					<th>Status</th>
					<th>Rate Limit</th>
					<th>Enabled</th>
				</tr>
			</thead>
			<tbody>
				{#each sortedTokens as token}
					{@const account = accounts.find((account) => account.id === token.account)}
					{@const client = clients.find((client) => client.id === token.client)}
					{@const ratelimitDetails = ratelimits?.[token.id]}

					{#if account && client && ratelimitDetails}
						<TokensTableRow {token} {account} {client} {ratelimitDetails} />
					{/if}
				{:else}
					<tr>
						<td colspan="5" class="text-center py-4">No tokens found.</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
</div>
