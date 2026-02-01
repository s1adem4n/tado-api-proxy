<script lang="ts">
	import { pb, type Account, type Client, type RatelimitDetails, type Token } from '@/lib/pb';

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

	let loading = $state(false);
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
	<td>
		<div class="flex justify-center">
			<input
				class="checkbox checkbox-neutral"
				type="checkbox"
				checked={!token.disabled}
				onchange={async () => {
					if (loading) return;
					loading = true;
					try {
						await pb
							.collection('tokens')
							.update(token.id, { disabled: token.disabled ? false : true });
					} finally {
						loading = false;
					}
				}}
			/>
		</div>
	</td>
</tr>
