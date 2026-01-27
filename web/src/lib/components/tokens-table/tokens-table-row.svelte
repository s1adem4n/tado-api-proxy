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
	<td class="text-base-content/70">{formatLastUsed(token.used)}</td>
	<td>
		<span
			class="badge badge-sm capitalize"
			class:badge-success={token.status === 'valid'}
			class:badge-error={token.status === 'invalid'}
		>
			{token.status}
		</span>
	</td>
	<td>
		<div class="flex flex-col gap-1">
			<progress
				class="progress-sm progress w-16"
				value={ratelimitDetails.used}
				max={ratelimitDetails.limit}
			></progress>
			<span class="text-sm text-base-content/70">
				{ratelimitDetails.used}/{ratelimitDetails.limit}
			</span>
		</div>
	</td>
</tr>
