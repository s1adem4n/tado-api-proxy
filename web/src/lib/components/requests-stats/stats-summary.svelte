<script lang="ts">
	import type { Requests } from '@/lib/pb';

	let { requests }: { requests: Requests[] } = $props();

	const totalRequests = $derived(requests.length);
	const successfulRequests = $derived(
		requests.filter((r) => r.status >= 200 && r.status < 400).length
	);
	const failedRequests = $derived(requests.filter((r) => r.status >= 400).length);
	const successRate = $derived(
		totalRequests > 0 ? Math.floor((successfulRequests / totalRequests) * 100) : 0
	);
</script>

<div class="grid grid-cols-2 gap-4 sm:grid-cols-4">
	<div class="stat rounded-box border border-base-content/10 bg-base-100 p-4">
		<div class="stat-title text-sm">Total Requests</div>
		<div class="stat-value text-2xl">{totalRequests}</div>
	</div>

	<div class="stat rounded-box border border-base-content/10 bg-base-100 p-4">
		<div class="stat-title text-sm">Successful</div>
		<div class="stat-value text-2xl text-success">{successfulRequests}</div>
	</div>

	<div class="stat rounded-box border border-base-content/10 bg-base-100 p-4">
		<div class="stat-title text-sm">Failed</div>
		<div class="stat-value text-2xl text-error">{failedRequests}</div>
	</div>

	<div class="stat rounded-box border border-base-content/10 bg-base-100 p-4">
		<div class="stat-title text-sm">Success Rate</div>
		<div class="stat-value text-2xl">{successRate}%</div>
	</div>
</div>
