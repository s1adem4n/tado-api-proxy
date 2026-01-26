<script lang="ts">
	type TimeFrame = '1h' | '24h' | '7d';

	let { value, onChange }: { value: TimeFrame; onChange: (value: TimeFrame) => void } = $props();

	const options: { label: string; value: TimeFrame }[] = [
		{ label: 'Last Hour', value: '1h' },
		{ label: 'Last 24 Hours', value: '24h' },
		{ label: 'Last 7 Days', value: '7d' }
	];
</script>

<div class="flex gap-1">
	{#each options as option}
		<button
			class="btn btn-sm"
			class:btn-primary={value === option.value}
			class:btn-ghost={value !== option.value}
			onclick={() => onChange(option.value)}
		>
			{option.label}
		</button>
	{/each}
</div>
