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

<div class="card border border-base-content/5 bg-base-100">
	<div class="card-body">
		<p class="text-sm text-base-content/70">
			Please log in with your administrator account to manage the Tado API Proxy.
		</p>

		<form class="mt-2 flex flex-col gap-4" onsubmit={submit}>
			<div class="flex flex-col gap-2">
				<label for="email" class="label">Email</label>
				<input
					type="email"
					id="email"
					class="input w-full"
					placeholder="example@mail.com"
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
					bind:value={password}
				/>
			</div>

			<button type="submit" class="btn btn-primary" disabled={!email || !password}>Login</button>

			{#if error}
				<div class="alert alert-error">
					<span>{error}</span>
				</div>
			{/if}
		</form>
	</div>
</div>
