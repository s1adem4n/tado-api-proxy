<script lang="ts">
	import {
		TimeFrameSelector,
		StatsSummary,
		RequestsTable,
		EndpointStats
	} from '@/lib/components/requests-stats';
	import { pb } from '@/lib/pb';
	import { MultipleSubscription, navigation } from '@/lib/stores.svelte';
	import ArrowLeftIcon from '~icons/lucide/arrow-left';
	import LogOutIcon from '~icons/lucide/log-out';

	type TimeFrame = '1h' | '24h' | '7d';
	const validTimeFrames: TimeFrame[] = ['1h', '24h', '7d'];

	let timeFrame = $derived.by(() => {
		const tf = navigation.getQuery('tf', '24h');
		return validTimeFrames.includes(tf as TimeFrame) ? (tf as TimeFrame) : '24h';
	});

	function setTimeFrame(tf: TimeFrame) {
		navigation.setQuery('tf', tf);
	}

	function getFilterDate(tf: TimeFrame): string {
		const now = new Date();
		switch (tf) {
			case '1h':
				now.setHours(now.getHours() - 1);
				break;
			case '24h':
				now.setDate(now.getDate() - 1);
				break;
			case '7d':
				now.setDate(now.getDate() - 7);
				break;
		}
		return now.toISOString().replace('T', ' ');
	}

	const requests = new MultipleSubscription(
		pb.collection('requests'),
		() => `created >= "${getFilterDate(timeFrame)}"`
	);
	const tokens = new MultipleSubscription(pb.collection('tokens'));
	const accounts = new MultipleSubscription(pb.collection('accounts'));
</script>

<header class="flex items-center justify-between border-b border-base-content/5 pb-2">
	<div class="flex items-center gap-2">
		<button
			class="btn btn-square btn-ghost btn-sm"
			onclick={() => navigation.navigate('/')}
			title="Back to Home"
		>
			<ArrowLeftIcon class="h-4 w-4" />
		</button>
		<h1 class="text-2xl font-semibold sm:text-3xl">Request Statistics</h1>
	</div>

	<button class="btn btn-ghost btn-sm" onclick={() => pb.authStore.clear()}>
		<LogOutIcon class="h-4 w-4" />
		Logout
	</button>
</header>

<TimeFrameSelector value={timeFrame} onChange={setTimeFrame} />

<StatsSummary requests={requests.items} />

<div class="flex flex-col gap-2">
	<h3 class="text-lg font-medium">Endpoints</h3>
	<EndpointStats requests={requests.items} />
</div>

<div class="flex flex-col gap-2">
	<h3 class="text-lg font-medium">Recent Requests</h3>
	<RequestsTable requests={requests.items} tokens={tokens.items} accounts={accounts.items} />
</div>
