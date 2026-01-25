<script lang="ts">
	import { pb } from '@/lib/pb';

	let email = $state('');
	let password = $state('');
	let error = $state('');

	async function submit(e: Event) {
		e.preventDefault();

		try {
			await pb.collection('_superusers').authWithPassword(email, password);
			error = '';
		} catch (err) {
			error = 'Login failed. Please check your credentials.';
		}
	}
</script>

<p class="text-sm">Please log in with your administrator account to manage the Tado API Proxy.</p>

<form class="flex flex-col gap-4" onsubmit={submit}>
	<div class="flex flex-col gap-2">
		<label for="email" class="label">Email</label>
		<input type="email" class="input w-full" placeholder="example@mail.com" bind:value={email} />
	</div>

	<div class="flex flex-col gap-2">
		<label for="password" class="label">Password</label>
		<input type="password" class="input w-full" placeholder="Your password" bind:value={password} />
	</div>

	<button type="submit" class="btn" disabled={!email || !password}>Login</button>

	{#if error}
		<p class="text-error">{error}</p>
	{/if}
</form>
