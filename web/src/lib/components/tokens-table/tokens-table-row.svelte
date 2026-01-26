<script lang="ts">
	import type { Account, Client, RatelimitDetails, Token } from '@/lib/pb';

	let {
		token,
		account,
		client,
		ratelimitDetails
	}: {
		token: Token;
		account: Account;
		client: Client;
		ratelimitDetails: RatelimitDetails;
	} = $props();

	function formatLastUsed(used: string) {
		if (!used) return 'Never';

		// if on same day, show only time
		const usedDate = new Date(used);
		const now = new Date();

		if (usedDate.toDateString() === now.toDateString()) {
			return usedDate.toLocaleTimeString();
		}
		return usedDate.toLocaleString();
	}
</script>

<tr>
	<td>{account.email}</td>
	<td>{client.name}</td>
	<td>{formatLastUsed(token.used)}</td>
	<td class="capitalize">{token.status}</td>
	<td>{ratelimitDetails.used}/{ratelimitDetails.limit}</td>
</tr>
