<script lang="ts">
	import TokensTableRow from './tokens-table-row.svelte';
	import type { Account, Client, Token } from '@/lib/pb';

	let {
		tokens,
		accounts,
		clients
	}: {
		tokens: Token[];
		accounts: Account[];
		clients: Client[];
	} = $props();

	const sortedTokens = $derived(tokens.toSorted((a, b) => b.used.localeCompare(a.used)));
</script>

<div class="flex flex-col gap-2">
	<h2 class="text-2xl font-semibold">Tokens</h2>

	<div class="overflow-x-auto rounded-box border border-base-content/5 bg-base-100">
		<table class="table">
			<thead>
				<tr>
					<th>Account</th>
					<th>Client</th>
					<th>Last used</th>
					<th>Status</th>
				</tr>
			</thead>
			<tbody>
				{#each sortedTokens as token}
					{@const account = accounts.find((account) => account.id === token.account)}
					{@const client = clients.find((client) => client.id === token.client)}

					{#if account && client}
						<TokensTableRow {token} {account} {client} />
					{/if}
				{:else}
					<tr>
						<td colspan="4" class="text-center py-4">No tokens found.</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
</div>
