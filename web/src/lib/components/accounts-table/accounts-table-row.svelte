<script lang="ts">
	import { pb, type Account, type Home } from '@/lib/pb';
	import TrashIcon from '~icons/lucide/trash';

	let { account, homes }: { account: Account; homes: Home[] } = $props();

	let loading = $state(false);
	let deleteDialog: HTMLDialogElement;
</script>

<tr>
	<td>{account.email}</td>
	<td>
		<ul>
			{#each homes as home}
				<li>{home?.name}</li>
			{/each}
		</ul>
	</td>
	<td>
		<button
			class="btn btn-square btn-ghost btn-sm btn-error"
			onclick={() => deleteDialog.showModal()}
		>
			<TrashIcon class="h-4 w-4" />
		</button>
	</td>
</tr>

<dialog class="modal" bind:this={deleteDialog}>
	<div class="modal-box">
		<h3 class="text-lg font-bold">Are you sure you want to delete this account?</h3>
		<p class="py-4">This action cannot be undone.</p>

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
				Delete
			</button>
		</div>
	</div>
</dialog>
