<script lang="ts">
	import type { Requests } from '@/lib/pb';

	let { requests }: { requests: Requests[] } = $props();

	interface EndpointStat {
		endpoint: string;
		total: number;
		successful: number;
		failed: number;
		successRate: number;
	}

	const endpointStats = $derived.by(() => {
		const stats = new Map<string, { total: number; successful: number; failed: number }>();

		for (const req of requests) {
			const endpoint = extractEndpoint(req.url);
			const current = stats.get(endpoint) ?? { total: 0, successful: 0, failed: 0 };

			current.total++;
			if (req.status >= 200 && req.status < 400) {
				current.successful++;
			} else {
				current.failed++;
			}

			stats.set(endpoint, current);
		}

		const result: EndpointStat[] = [];
		stats.forEach((value, key) => {
			result.push({
				endpoint: key,
				total: value.total,
				successful: value.successful,
				failed: value.failed,
				successRate: Math.round((value.successful / value.total) * 100)
			});
		});

		return result.sort((a, b) => b.total - a.total);
	});

	function extractEndpoint(url: string): string {
		try {
			const parsed = new URL(url);
			// Normalize path by replacing IDs with placeholders
			return parsed.pathname.replace(/\/\d+/g, '/:id').replace(/\/[a-f0-9-]{36}/gi, '/:uuid');
		} catch {
			return url;
		}
	}
</script>

<div class="overflow-x-auto rounded-box border border-base-content/5 bg-base-100">
	<table class="table table-sm">
		<thead>
			<tr>
				<th>Endpoint</th>
				<th class="text-right">Total</th>
				<th class="text-right">OK</th>
				<th class="text-right">Failed</th>
				<th class="text-right">Success</th>
			</tr>
		</thead>
		<tbody>
			{#each endpointStats.slice(0, 20) as stat}
				<tr>
					<td class="font-mono text-sm">{stat.endpoint}</td>
					<td class="text-right">{stat.total}</td>
					<td class="text-right text-success">{stat.successful}</td>
					<td class="text-right text-error">{stat.failed}</td>
					<td class="text-right">
						<span
							class="badge badge-sm"
							class:badge-success={stat.successRate >= 90}
							class:badge-warning={stat.successRate >= 70 && stat.successRate < 90}
							class:badge-error={stat.successRate < 70}
						>
							{stat.successRate}%
						</span>
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

{#if endpointStats.length > 20}
	<p class="text-center text-sm text-base-content/50">
		Showing top 20 of {endpointStats.length} endpoints
	</p>
{/if}
