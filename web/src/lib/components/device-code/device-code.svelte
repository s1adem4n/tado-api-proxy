<script lang="ts">
	import { pb, type Client, type Code } from '@/lib/pb';

	let {
		clients,
		codes
	}: {
		clients: Client[];
		codes: Code[];
	} = $props();

	let now = $state(new Date());
	$effect(() => {
		const interval = setInterval(() => {
			now = new Date();
		}, 1000);

		return () => clearInterval(interval);
	});

	const supportedClients = $derived(clients.filter((client) => client.type === 'deviceCode'));
	const currentCode = $derived(
		codes.toSorted((a, b) => new Date(b.expires).getTime() - new Date(a.expires).getTime()).at(0)
	);

	let status = $state<'idle' | 'loading' | 'error'>('idle');

	let client = $state('');
	$effect(() => {
		if (supportedClients.length > 0 && !client) {
			client = supportedClients[0].id;
		}
	});

	async function submit(e: Event) {
		e.preventDefault();
		status = 'loading';

		// create a new window before the await to avoid popup blockers
		const authWindow = window.open('', '_blank');

		try {
			const code = await pb.collection('codes').create({
				client
			});

			if (authWindow) {
				authWindow.location.href = code.verificationURI;
			}
		} catch (err) {
			status = 'error';
		} finally {
			status = 'idle';
		}
	}
</script>

<div class="flex flex-col gap-2">
	<h2 class="text-2xl font-semibold">Authorize Official API</h2>
	<p class="mb-2 text-sm text-base-content/70">
		You can only authorize accounts that you have already added in the table above.
	</p>

	<form class="flex flex-col gap-4" onsubmit={submit}>
		{#if supportedClients.length > 1}
			<div class="flex flex-col gap-2">
				<label for="client" class="label">Select Client</label>
				<select id="client" class="select w-full" bind:value={client}>
					{#each supportedClients as c}
						<option value={c.id}>{c.name}</option>
					{/each}
				</select>
			</div>
		{/if}

		<button
			type="submit"
			class="btn btn-primary"
			disabled={status === 'loading' || currentCode?.status === 'pending'}
		>
			{#if status === 'loading'}
				<span class="loading loading-spinner"></span>
				Starting Authorization...
			{:else if currentCode?.status === 'pending'}
				<span class="loading loading-spinner"></span>
				Waiting for Authorization...
			{:else}
				Start Authorization
			{/if}
		</button>

		{#if currentCode?.status === 'pending'}
			<p class="text-sm text-success">
				An authorization is already in progress. Please complete it in the opened window or <a
					href={currentCode.verificationURI}
					target="_blank"
					class="underline"
				>
					open it again
				</a>.
			</p>
		{:else if currentCode?.status === 'expired'}
			<p class="text-sm text-error">
				The previous authorization has expired. Please start a new authorization.
			</p>
		{:else if currentCode?.status === 'unknownAccount'}
			<p class="text-sm text-error">
				The account you tried to authorize does not exist in the system. Please add the account
				first.
			</p>
		{/if}

		{#if status === 'error'}
			<p class="text-sm text-error">
				An error occurred while starting the authorization. Please try again.
			</p>
		{/if}
	</form>
</div>
