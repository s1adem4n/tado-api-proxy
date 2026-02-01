<script lang="ts">
	import { pb } from '@/lib/pb';
	import { MultipleSubscription } from '@/lib/stores.svelte';
	import AlertTriangleIcon from '~icons/lucide/alert-triangle';
	import CopyIcon from '~icons/lucide/copy';
	import ShieldCheckIcon from '~icons/lucide/shield-check';

	const settingsSub = new MultipleSubscription(pb.collection('settings'));
	let settings = $derived(settingsSub.items[0]);

	async function toggleProtection() {
		if (!settings) return;
		await pb.collection('settings').update(settings.id, {
			proxyTokenEnabled: !settings.proxyTokenEnabled
		});
	}

	function getEndpoint(token: string) {
		return `${window.location.origin}/${token}`;
	}

	function copyEndpoint() {
		if (settings) {
			navigator.clipboard.writeText(getEndpoint(settings.proxyToken));
		}
	}
</script>

<div class="flex flex-col gap-2">
	<h2 class="text-2xl font-semibold">Proxy Access</h2>
	<div class="flex flex-col gap-4 rounded-box border border-base-content/5 bg-base-100 p-4">
		{#if settings}
			{#if !settings.proxyTokenEnabled}
				<div role="alert" class="alert alert-vertical alert-warning sm:alert-horizontal">
					<AlertTriangleIcon class="h-5 w-5" />
					<div class="flex-1">
						<h3 class="font-bold">Unauthenticated Access Enabled</h3>
						<div class="text-xs">
							The <code>/api/v2</code> endpoint is open. Anyone in your network who can reach the proxy
							can make authenticated requests to tado.
						</div>
					</div>
					<button class="btn btn-sm" onclick={toggleProtection}
						>Disable Unauthenticated Access</button
					>
				</div>
			{:else}
				<div role="alert" class="alert alert-vertical alert-success sm:alert-horizontal">
					<ShieldCheckIcon class="h-5 w-5" />
					<div class="flex-1">
						<h3 class="font-bold">Protected Mode Enabled</h3>
						<div class="text-xs">
							Only requests with the correct token in the URL path are accepted.
						</div>
					</div>
					<button class="btn btn-sm" onclick={toggleProtection}>
						Enable Unauthenticated Access
					</button>
				</div>

				<div class="flex flex-col gap-2">
					<label class="label" for="proxy-endpoint-input">
						<span class="label-text font-medium">Authenticated Base URL</span>
					</label>
					<div class="join w-full">
						<input
							type="text"
							readonly
							value={getEndpoint(settings.proxyToken)}
							id="proxy-endpoint-input"
							class="input-bordered input join-item w-full font-mono text-sm"
						/>
						<button
							class="btn join-item btn-square"
							onclick={copyEndpoint}
							aria-label="Copy Endpoint"
						>
							<CopyIcon />
						</button>
					</div>
					<span class="text-sm text-base-content/70">
						Use this base URL for your tado client configuration.
					</span>
				</div>
			{/if}
		{:else}
			<div>Loading settings...</div>
		{/if}
	</div>
</div>
