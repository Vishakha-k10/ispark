<script lang="ts">
	import { fade } from 'svelte/transition';
	import { onMount } from 'svelte';
	import { API_BASE_URL } from '$lib/config';

	// Types
	interface AdminProfile {
		admin_id: string;
		name: string;
		email: string;
		role: string;
		assigned_batch: string;
		created_at: string;
		updated_at: string;
	}

	// Props
	let {
		admin,
		onSave,
		onCancel
	}: {
		admin: AdminProfile | null;
		onSave: (
			updatedAdmin: AdminProfile,
			stats: {
				assigned_students: number;
				verified_certificates: number;
				pending_reviews: number;
				supervised_activities: number;
			} | null
		) => void;
		onCancel: () => void;
	} = $props();

	// Form State
	let name = $state('');
	let email = $state('');
	let submitting = $state(false);
	let errorMsg = $state<string | null>(null);
	let successMsg = $state<string | null>(null);

	// Basic email format check
	const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

	onMount(() => {
		if (admin) {
			name = admin.name;
			email = admin.email;
		}
	});

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		errorMsg = null;
		successMsg = null;

		// Trim fields
		name = name.trim();
		email = email.trim();

		// Reject empty or invalid fields
		if (!name) {
			errorMsg = 'Name is required';
			return;
		}

		if (!email) {
			errorMsg = 'Email is required';
			return;
		}

		if (!emailRegex.test(email)) {
			errorMsg = 'Please enter a valid email address';
			return;
		}

		submitting = true;

		const token = localStorage.getItem('admin_token');
		if (!token) {
			errorMsg = 'Authentication token missing. Please log in again.';
			submitting = false;
			return;
		}

		try {
			const res = await fetch(`${API_BASE_URL}/api/admin/profile`, {
				method: 'PUT',
				headers: {
					'Content-Type': 'application/json',
					Authorization: `Bearer ${token}`
				},
				body: JSON.stringify({
					name,
					email
				})
			});

			const data = await res.json();

			if (!res.ok) {
				if (res.status === 409) {
					throw new Error('Email already exists. Please use a different email.');
				}
				throw new Error(data.error || 'Failed to update profile');
			}

			successMsg = 'Profile updated successfully!';

			// Notify parent layout about updated data
			if (data.admin) {
				onSave(data.admin, data.stats);
			}

			setTimeout(() => {
				onCancel();
			}, 1000);
		} catch (err) {
			errorMsg = err instanceof Error ? err.message : 'Failed to update profile';
		} finally {
			submitting = false;
		}
	}
</script>

<div class="space-y-3 font-sans" transition:fade={{ duration: 150 }}>

	<!-- Form Card -->
	<div class="bg-white rounded-2xl border border-slate-200 p-6 sm:p-7 shadow-sm max-w-2xl">
		<form onsubmit={handleSubmit} class="space-y-4">
			{#if errorMsg}
				<div
					class="p-3.5 bg-rose-50 border border-rose-200 text-rose-700 text-xs font-semibold rounded-lg"
					transition:fade={{ duration: 150 }}
				>
					{errorMsg}
				</div>
			{/if}
			{#if successMsg}
				<div
					class="p-3.5 bg-emerald-50 border border-emerald-200 text-emerald-700 text-xs font-semibold rounded-lg"
					transition:fade={{ duration: 150 }}
				>
					{successMsg}
				</div>
			{/if}

			<!-- Admin ID (Read-only) -->
			<div class="flex flex-col gap-1.5">
				<label for="admin-id" class="text-[10px] font-bold text-slate-450 uppercase tracking-wider">
					Admin ID
				</label>
				<input
					id="admin-id"
					type="text"
					value={admin?.admin_id ?? ''}
					disabled
					class="w-full px-3.5 py-2.5 bg-slate-50 border border-slate-200 rounded-lg text-sm text-slate-400 font-medium select-none focus:outline-none"
				/>
			</div>

			<!-- Full Name -->
			<div class="flex flex-col gap-1.5">
				<label
					for="edit-name"
					class="text-[10px] font-bold text-slate-450 uppercase tracking-wider"
				>
					Full Name
				</label>
				<input
					id="edit-name"
					type="text"
					bind:value={name}
					disabled={submitting}
					placeholder="Enter your full name"
					class="w-full px-3.5 py-2.5 bg-white border border-slate-200 focus:border-[#881B1B]/50 focus:ring-1 focus:ring-[#881B1B]/40 rounded-lg text-sm focus:outline-none disabled:bg-slate-50 disabled:text-slate-400 transition-colors"
					required
				/>
			</div>

			<!-- Email Address -->
			<div class="flex flex-col gap-1.5">
				<label
					for="edit-email"
					class="text-[10px] font-bold text-slate-450 uppercase tracking-wider"
				>
					Email Address
				</label>
				<input
					id="edit-email"
					type="email"
					bind:value={email}
					disabled={submitting}
					placeholder="Enter your email address"
					class="w-full px-3.5 py-2.5 bg-white border border-slate-200 focus:border-[#881B1B]/50 focus:ring-1 focus:ring-[#881B1B]/40 rounded-lg text-sm focus:outline-none disabled:bg-slate-50 disabled:text-slate-400 transition-colors"
					required
				/>
			</div>

			<!-- Form Actions -->
			<div class="flex items-center justify-end gap-3 pt-3 border-t border-slate-100">
				<button
					type="button"
					onclick={onCancel}
					disabled={submitting}
					class="px-4 py-2 border border-slate-250 hover:bg-slate-50 disabled:opacity-50 text-slate-700 bg-white rounded-lg text-xs font-bold transition-colors cursor-pointer focus:outline-none shadow-3xs"
				>
					Cancel
				</button>
				<button
					type="submit"
					disabled={submitting}
					class="inline-flex items-center justify-center gap-1.5 px-4 py-2 bg-[#881B1B] hover:bg-[#881B1B]/90 disabled:bg-[#881B1B]/50 text-white rounded-lg text-xs font-bold transition-colors cursor-pointer focus:outline-none shadow-3xs"
				>
					{#if submitting}
						<svg
							class="animate-spin -ml-1 mr-1.5 h-3.5 w-3.5 text-white"
							fill="none"
							viewBox="0 0 24 24"
						>
							<circle
								class="opacity-25"
								cx="12"
								cy="12"
								r="10"
								stroke="currentColor"
								stroke-width="4"
							></circle>
							<path
								class="opacity-75"
								fill="currentColor"
								d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
							></path>
						</svg>
						Saving...
					{:else}
						Save Changes
					{/if}
				</button>
			</div>
		</form>
	</div>
</div>