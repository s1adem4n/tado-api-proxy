<script lang="ts">
	import { pb, type Client, type Code } from '@/lib/pb';

	let {
		clients,
		codes
	}: {
		clients: Client[];
		codes: Code[];
	} = $props();

	const supportedClients = $derived(clients.filter((client) => client.type === 'deviceCode'));

	let loading = $state(false);
	let error = $state('');

	let client = $state('');
	$effect(() => {
		if (supportedClients.length > 0 && !client) {
			client = supportedClients[0].id;
		}
	});

	async function submit(e: Event) {
		e.preventDefault();
		loading = true;

		try {
			await pb.collection('codes').create({
				client
			});
		} catch (err) {
			error = 'Failed to create device code authorization.';
		} finally {
			loading = false;
		}
	}

	let now = $state(new Date());
	$effect(() => {
		const interval = setInterval(() => {
			now = new Date();
		}, 1000);

		return () => clearInterval(interval);
	});

	const validCodes = $derived(codes.filter((code) => new Date(code.expires) > now && !code.token));
</script>

<div class="flex flex-col gap-2">
	<h2 class="text-2xl font-semibold">Authorize Official API</h2>

	<div class="flex flex-col gap-4 rounded-box border border-base-content/5 bg-base-100 p-4">
		<form class="flex flex-col gap-4" onsubmit={submit}>
			<div class="flex flex-col gap-2">
				<label for="client" class="label">Select Client</label>
				<select id="client" class="select w-full" bind:value={client}>
					{#each supportedClients as c}
						<option value={c.id}>{c.name}</option>
					{/each}
				</select>
			</div>

			<button type="submit" class="btn btn-primary" disabled={loading}>
				{#if loading}
					<span class="loading loading-spinner"></span>
				{/if}
				Authorize
			</button>

			{#if error}
				<p class="text-error">{error}</p>
			{/if}
		</form>

		{#each validCodes as code}
			<a
				href={code.verificationURI}
				target="_blank"
				rel="noopener noreferrer"
				class="btn w-full btn-outline"
			>
				Authorize {code.userCode} &rarr;
			</a>
		{/each}
	</div>
</div>
