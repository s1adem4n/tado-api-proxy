<script lang="ts">
	import { pb, type Account, type Home } from '@/lib/pb';
	import TrashIcon from '~icons/lucide/trash';

	let {
		index,
		total,
		account,
		homes
	}: { index: number; total: number; account: Account; homes: Home[] } = $props();

	let loading = $state(false);
	let deleteDialog: HTMLDialogElement;
</script>

<tr class={index === total - 1 ? '*:border-b-0' : ''}>
	<td class="font-medium">{account.email}</td>
	<td>
		{#if homes.length > 0}
			<div class="flex flex-wrap gap-1">
				{#each homes as home}
					<span class="badge badge-ghost badge-sm">{home?.name}</span>
				{/each}
			</div>
		{:else}
			<span class="text-sm text-base-content/50">No homes</span>
		{/if}
	</td>
	<td>
		<button
			class="btn btn-square btn-ghost btn-sm btn-error"
			onclick={() => deleteDialog.showModal()}
			title="Delete account"
		>
			<TrashIcon class="h-4 w-4" />
		</button>
	</td>
</tr>

<dialog class="modal" bind:this={deleteDialog}>
	<div class="modal-box">
		<h3 class="text-lg font-bold">Delete Account</h3>
		<p class="py-4 text-base-content/70">
			Are you sure you want to delete <strong class="text-base-content">{account.email}</strong>?
			This action cannot be undone.
		</p>

		<div class="modal-action">
			<button class="btn" onclick={() => deleteDialog.close()}>Cancel</button>
			<button
				class="btn btn-error"
				disabled={loading}
				onclick={async () => {
					loading = true;
					await pb.collection('accounts').delete(account.id);
					deleteDialog.close();
					loading = false;
				}}
			>
				{#if loading}
					<span class="loading loading-sm loading-spinner"></span>
				{/if}
				Delete
			</button>
		</div>
	</div>
</dialog>
