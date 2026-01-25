<script lang="ts">
	import AccountsTableRow from './accounts-table-row.svelte';
	import { pb, type Account, type Home } from '@/lib/pb';
	import PlusIcon from '~icons/lucide/plus';

	let { accounts, homes }: { accounts: Account[]; homes: Home[] } = $props();

	function getAccountHomes(account: Account): Home[] {
		return account.homes
			.map((id) => homes.find((home) => home.id === id))
			.filter(Boolean) as Home[];
	}

	let addAccountDialog: HTMLDialogElement;

	let loading = $state(false);
	let error = $state('');

	let email = $state('');
	let password = $state('');

	async function submit(e: Event) {
		e.preventDefault();
		loading = true;

		try {
			await pb.collection('accounts').create({
				email,
				password,
				homes: []
			});
			addAccountDialog.close();
		} catch (err) {
			error = 'Failed to add account. Please check your credentials and try again.';
		} finally {
			loading = false;
		}
	}
</script>

<div class="flex flex-col gap-2">
	<div class="flex items-center justify-between">
		<h2 class="text-2xl font-semibold">Accounts</h2>

		<button class="btn btn-sm" onclick={() => addAccountDialog.showModal()}>
			<PlusIcon class="mr-2 h-4 w-4" />
			Add an Account
		</button>
	</div>

	<div class="overflow-x-auto rounded-box border border-base-content/5 bg-base-100">
		<table class="table">
			<thead>
				<tr>
					<th>Email</th>
					<th>Homes</th>
					<th class="w-0">
						<span class="sr-only">Actions</span>
					</th>
				</tr>
			</thead>
			<tbody>
				{#each accounts as account}
					<AccountsTableRow {account} homes={getAccountHomes(account)} />
				{:else}
					<tr>
						<td colspan="3" class="text-center py-4">No accounts found.</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
</div>

<dialog class="modal" bind:this={addAccountDialog}>
	<div class="modal-box">
		<h3 class="text-lg font-bold">Add new Account</h3>

		<form class="mt-4 flex flex-col gap-4" onsubmit={submit}>
			<div class="flex flex-col gap-2">
				<label for="email" class="label">Email</label>
				<input
					type="email"
					id="email"
					class="input w-full"
					placeholder="your@email.com"
					required
					bind:value={email}
				/>
			</div>

			<div class="flex flex-col gap-2">
				<label for="password" class="label">Password</label>
				<input
					type="password"
					id="password"
					class="input w-full"
					placeholder="Your password"
					required
					bind:value={password}
				/>
			</div>

			{#if error}
				<p class="text-error">{error}</p>
			{/if}

			<div class="modal-action">
				<button type="button" class="btn" onclick={() => addAccountDialog.close()}>Close</button>

				<button type="submit" class="btn btn-primary" disabled={loading}>
					{#if loading}
						<span class="loading loading-spinner"></span>
					{/if}
					Add Account
				</button>
			</div>
		</form>
	</div>
</dialog>
