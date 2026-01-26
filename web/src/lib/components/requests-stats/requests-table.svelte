<script lang="ts">
	import type { Requests, Token, Account } from '@/lib/pb';

	let {
		requests,
		tokens,
		accounts
	}: {
		requests: Requests[];
		tokens: Token[];
		accounts: Account[];
	} = $props();

	const sortedRequests = $derived(
		requests.toSorted((a, b) => new Date(b.created).getTime() - new Date(a.created).getTime())
	);

	function getAccountEmail(tokenId: string): string {
		const token = tokens.find((t) => t.id === tokenId);
		if (!token) return 'Unknown';
		const account = accounts.find((a) => a.id === token.account);
		return account?.email ?? 'Unknown';
	}

	function formatTime(dateStr: string): string {
		const date = new Date(dateStr);
		const now = new Date();

		if (date.toDateString() === now.toDateString()) {
			return date.toLocaleTimeString();
		}
		return date.toLocaleString();
	}

	function getStatusBadgeClass(status: number): string {
		if (status >= 200 && status < 300) return 'badge-success';
		if (status >= 300 && status < 400) return 'badge-warning';
		return 'badge-error';
	}

	function shortenUrl(url: string): string {
		try {
			const parsed = new URL(url);
			return parsed.pathname;
		} catch {
			return url;
		}
	}
</script>

<div class="overflow-x-auto rounded-box border border-base-content/5 bg-base-100">
	<table class="table table-sm">
		<thead>
			<tr>
				<th>Time</th>
				<th>Account</th>
				<th>Method</th>
				<th>URL</th>
				<th>Status</th>
			</tr>
		</thead>
		<tbody>
			{#each sortedRequests.slice(0, 100) as request}
				<tr>
					<td class="whitespace-nowrap text-base-content/70">{formatTime(request.created)}</td>
					<td class="max-w-32 truncate">{getAccountEmail(request.token)}</td>
					<td>
						<span class="badge badge-ghost font-mono badge-sm">{request.method}</span>
					</td>
					<td class="max-w-48 truncate font-mono text-sm" title={request.url}>
						{shortenUrl(request.url)}
					</td>
					<td>
						<span class="badge badge-sm {getStatusBadgeClass(request.status)}"
							>{request.status}</span
						>
					</td>
				</tr>
			{:else}
				<tr>
					<td colspan="5" class="py-8 text-center text-base-content/70">
						No requests found in this time frame.
					</td>
				</tr>
			{/each}
		</tbody>
	</table>
</div>

{#if sortedRequests.length > 100}
	<p class="text-center text-sm text-base-content/50">
		Showing first 100 of {sortedRequests.length} requests
	</p>
{/if}
