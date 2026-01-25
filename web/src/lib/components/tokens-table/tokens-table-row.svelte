<script lang="ts">
	import type { Account, Client, Token } from '@/lib/pb';

	let {
		token,
		account,
		client
	}: {
		token: Token;
		account: Account;
		client: Client;
	} = $props();
</script>

<tr>
	<td>{account.email}</td>
	<td>{client.name}</td>
	<td>{token.used ? new Date(token.used).toLocaleString() : 'Never'}</td>
	<td class="capitalize">{token.status}</td>
</tr>
