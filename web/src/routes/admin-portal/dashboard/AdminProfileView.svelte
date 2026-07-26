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

	interface BatchInfo {
		batch: string;
		course: string;
		semester: string;
		studentCount: number;
	}

	// Props using Svelte 5 runes
	let {
		admin,
		loading,
		error,
		stats,
		onEditProfile
	}: {
		admin: AdminProfile | null;
		loading: boolean;
		error: string | null;
		stats: {
			assigned_students: number;
			verified_certificates: number;
			pending_reviews: number;
			supervised_activities: number;
		} | null;
		onEditProfile: () => void;
	} = $props();

	// Stats state using Svelte 5 runes
	let statsLoading = $state(true);
	let statsError = $state<string | null>(null);
	let assignedBatches = $state<BatchInfo[]>([]);

	// Change Password form state
	let isChangePasswordOpen = $state(false);
	let currentPassword = $state('');
	let newPassword = $state('');
	let confirmPassword = $state('');
	let isPasswordSubmitting = $state(false);
	let passwordError = $state<string | null>(null);
	let passwordSuccess = $state<string | null>(null);

	function getInitials(name: string): string {
		if (!name) return 'A';
		const parts = name.split(' ').filter((part) => {
			const lower = part.toLowerCase();
			return (
				lower !== 'dr.' &&
				lower !== 'prof.' &&
				lower !== 'mr.' &&
				lower !== 'ms.' &&
				lower !== 'mrs.'
			);
		});
		if (parts.length === 0) return 'A';
		if (parts.length === 1) return parts[0].substring(0, 2).toUpperCase();
		return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
	}

	function formatBatch(batch: string | null | undefined): string {
		if (!batch || batch.trim() === '') {
			return 'Not Assigned';
		}
		const batchRegex = /^([a-zA-Z]+)2K(\d{2})$/i;
		const match = batch.match(batchRegex);

		if (match) {
			const department = match[1].toUpperCase(); // "IT"
			const year = match[2]; // "24"
			return `${department} - Class of 20${year}`;
		}
		return batch.toUpperCase();
	}

	function openChangePassword() {
		currentPassword = '';
		newPassword = '';
		confirmPassword = '';
		passwordError = null;
		passwordSuccess = null;
		isChangePasswordOpen = true;
	}

	async function handleChangePassword(e: SubmitEvent) {
		e.preventDefault();
		if (!currentPassword || !newPassword || !confirmPassword) {
			passwordError = 'All fields are required';
			return;
		}
		if (newPassword !== confirmPassword) {
			passwordError = 'New passwords do not match';
			return;
		}

		const isMinLength = newPassword.length >= 8;
		const hasUppercase = /[A-Z]/.test(newPassword);
		const hasLowercase = /[a-z]/.test(newPassword);
		const hasNumber = /\d/.test(newPassword);
		const hasSpecial = /[^A-Za-z0-9]/.test(newPassword);

		if (!isMinLength || !hasUppercase || !hasLowercase || !hasNumber || !hasSpecial) {
			passwordError =
				'Password must be at least 8 characters long and contain at least one uppercase letter, one lowercase letter, one number, and one special character.';
			return;
		}

		isPasswordSubmitting = true;
		passwordError = null;
		passwordSuccess = null;

		const token = localStorage.getItem('admin_token');
		try {
			const res = await fetch(`${API_BASE_URL}/api/admin/change-password`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					Authorization: `Bearer ${token}`
				},
				body: JSON.stringify({
					current_password: currentPassword,
					new_password: newPassword,
					confirm_password: confirmPassword
				})
			});

			const data = await res.json();
			if (!res.ok) {
				throw new Error(data.error || 'Failed to change password');
			}

			passwordSuccess = 'Password changed successfully!';
			setTimeout(() => {
				isChangePasswordOpen = false;
			}, 1500);
		} catch (err) {
			passwordError = err instanceof Error ? err.message : 'An error occurred';
		} finally {
			isPasswordSubmitting = false;
		}
	}

	// Fetch dynamic stats from students list
	onMount(async () => {
		const token = localStorage.getItem('admin_token');
		if (!token) {
			statsLoading = false;
			return;
		}

		try {
			const res = await fetch(`${API_BASE_URL}/api/admin/students`, {
				headers: {
					Authorization: `Bearer ${token}`
				}
			});

			if (!res.ok) {
				throw new Error('Failed to fetch students stats');
			}

			const data = await res.json();
			const studentsList = data.students || [];

			const grouped: BatchInfo[] = [];

			for (const s of studentsList) {
				// Group by Batch / Course / Semester
				if (admin) {
					const batch = s.batch ?? admin.assigned_batch ?? 'Unknown';
					const course = s.course_name || '—';
					const semester = s.semester ? `Semester ${s.semester}` : '—';

					const existing = grouped.find(
						(g) => g.batch === batch && g.course === course && g.semester === semester
					);
					if (existing) {
						existing.studentCount++;
					} else {
						grouped.push({
							batch,
							course,
							semester,
							studentCount: 1
						});
					}
				}
			}

			assignedBatches = grouped;
		} catch (err) {
			console.error('Error fetching admin profile stats:', err);
			statsError = err instanceof Error ? err.message : 'Error loading overview data';
		} finally {
			statsLoading = false;
		}
	});
</script>

<div class="space-y-6 font-sans" transition:fade={{ duration: 150 }}>
	<!-- Profile Header Card -->
	<div class="bg-white rounded-2xl border border-slate-200 p-6 sm:p-8 shadow-sm relative">
		{#if loading}
			<!-- Loading State Skeleton -->
			<div class="flex flex-col md:flex-row items-center gap-6 animate-pulse">
				<div class="w-24 h-24 rounded-full bg-slate-200 shrink-0"></div>
				<div class="flex-grow space-y-3.5 w-full">
					<div class="h-6 bg-slate-200 rounded w-1/3"></div>
					<div class="grid grid-cols-1 sm:grid-cols-2 gap-y-3 gap-x-8">
						<div class="h-4 bg-slate-200 rounded w-3/4"></div>
						<div class="h-4 bg-slate-200 rounded w-3/4"></div>
						<div class="h-4 bg-slate-200 rounded w-2/3"></div>
						<div class="h-4 bg-slate-200 rounded w-2/3"></div>
						<div class="h-4 bg-slate-200 rounded w-1/2 sm:col-span-2"></div>
					</div>
				</div>
			</div>
		{:else if error}
			<!-- Error State -->
			<div class="p-6 text-center text-rose-600 bg-rose-50 border border-rose-100 rounded-lg">
				<svg
					xmlns="http://www.w3.org/2000/svg"
					fill="none"
					viewBox="0 0 24 24"
					stroke-width="2"
					stroke="currentColor"
					class="w-8 h-8 mx-auto mb-2 text-rose-500"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z"
					/>
				</svg>
				<h4 class="font-bold text-sm">Failed to Load Profile</h4>
				<p class="text-xs text-rose-500 mt-1">{error}</p>
			</div>
		{:else if admin}
			<!-- Main Profile Layout -->
			<div class="flex flex-col md:flex-row items-start gap-6">
				<!-- Left: Circular Avatar -->
				<div
					class="w-24 h-24 rounded-full bg-[#881B1B] text-white flex items-center justify-center font-bold text-3xl border-4 border-slate-100 shadow-md shrink-0 relative overflow-hidden font-serif select-none"
				>
					{getInitials(admin.name)}
				</div>

				<!-- Center: Info Fields -->
				<div class="flex-grow space-y-4 w-full">
					<div class="flex flex-col md:flex-row md:items-center justify-between gap-4">
						<h3 class="text-2xl font-bold text-slate-900 font-serif leading-none">
							{admin.name}
						</h3>

						<!-- Right: Actions Buttons -->
						<div class="flex gap-3 shrink-0">
							<!-- Edit Profile button -->
							<button
								type="button"
								onclick={onEditProfile}
								class="inline-flex items-center justify-center gap-1.5 px-4 py-2 border border-[#881B1B]/30 bg-white hover:bg-[#881B1B]/5 text-[#881B1B] rounded-lg text-xs font-bold transition-colors shadow-3xs cursor-pointer focus:outline-none"
							>
								<svg
									xmlns="http://www.w3.org/2000/svg"
									fill="none"
									viewBox="0 0 24 24"
									stroke-width="2.2"
									stroke="currentColor"
									class="w-3.5 h-3.5 text-[#881B1B]"
								>
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										d="m16.862 4.487 1.687-1.688a1.875 1.875 0 112.652 2.652L6.83 21.82a.75.75 0 01-.34.201L3 22.887l.859-3.542a.75.75 0 01.202-.34l11.758-11.76H16.862z"
									/>
								</svg>
								Edit Profile
							</button>

							<!-- Change Password button -->
							<button
								type="button"
								onclick={openChangePassword}
								class="inline-flex items-center justify-center gap-1.5 px-4 py-2 bg-[#881B1B] hover:bg-[#881B1B]/90 text-white rounded-lg text-xs font-bold transition-colors shadow-3xs cursor-pointer focus:outline-none"
							>
								<svg
									xmlns="http://www.w3.org/2000/svg"
									fill="none"
									viewBox="0 0 24 24"
									stroke-width="2.2"
									stroke="currentColor"
									class="w-3.5 h-3.5 text-white/90"
								>
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										d="M16.5 10.5V6.75a4.5 4.5 0 10-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 002.25-2.25v-6.75a2.25 2.25 0 00-2.25-2.25H6.75a2.25 2.25 0 00-2.25 2.25v6.75a2.25 2.25 0 002.25 2.25z"
									/>
								</svg>
								Change Password
							</button>
						</div>
					</div>

					<!-- Details Grid (Only fields backed by DB data) -->
					<div class="grid grid-cols-1 sm:grid-cols-2 gap-y-3.5 gap-x-8 text-xs leading-normal">
						<!-- Admin ID -->
						<div class="flex items-center gap-3">
							<svg
								xmlns="http://www.w3.org/2000/svg"
								fill="none"
								viewBox="0 0 24 24"
								stroke-width="2"
								stroke="currentColor"
								class="w-4 h-4 text-slate-400 shrink-0"
							>
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									d="M15 9h3.75M15 12h3.75M15 15h3.75M4.5 19.5h15a2.25 2.25 0 002.25-2.25V6.75A2.25 2.25 0 0019.5 4.5h-15a2.25 2.25 0 00-2.25 2.25v10.5A2.25 2.25 0 004.5 19.5zm6-10.125a1.875 1.875 0 11-3.75 0 1.875 1.875 0 013.75 0zm1.294 6.336a6.721 6.721 0 01-3.17.789 6.721 6.721 0 01-3.168-.789 3.376 3.376 0 016.338 0z"
								/>
							</svg>
							<div class="flex items-center gap-1.5">
								<span class="text-slate-500 font-semibold">Admin ID:</span>
								<span class="font-bold text-slate-800">{admin.admin_id}</span>
							</div>
						</div>

						<!-- Email -->
						<div class="flex items-center gap-3">
							<svg
								xmlns="http://www.w3.org/2000/svg"
								fill="none"
								viewBox="0 0 24 24"
								stroke-width="2"
								stroke="currentColor"
								class="w-4 h-4 text-slate-400 shrink-0"
							>
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									d="M21.75 6.75v10.5a2.25 2.25 0 0 1-2.25 2.25H4.5A2.25 2.25 0 0 1 2.25 17.25V6.75m19.5 0A2.25 2.25 0 0 0 19.5 4.5H4.5a2.25 2.25 0 0 0-2.25 2.25m19.5 0v.243a2.25 2.25 0 0 1-1.07 1.916l-7.5 4.615a2.25 2.25 0 0 1-2.36 0L3.32 8.91a2.25 2.25 0 0 1-1.07-1.916V6.75"
								/>
							</svg>
							<div class="flex items-center gap-1.5 min-w-0">
								<span class="text-slate-500 font-semibold">Email:</span>
								<span class="font-bold text-slate-800 truncate break-all">{admin.email}</span>
							</div>
						</div>

						<!-- Role -->
						<div class="flex items-center gap-3">
							<svg
								xmlns="http://www.w3.org/2000/svg"
								fill="none"
								viewBox="0 0 24 24"
								stroke-width="2"
								stroke="currentColor"
								class="w-4 h-4 text-slate-400 shrink-0"
							>
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									d="M9 12.75 11.25 15 15 9.75m-3-7.036A11.959 11.959 0 013.598 6 11.99 11.99 0 003 9.749c0 5.592 3.824 10.29 9 11.623 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285z"
								/>
							</svg>
							<div class="flex items-center gap-1.5">
								<span class="text-slate-500 font-semibold">Role:</span>
								<span class="font-bold text-slate-800 capitalize"
									>{admin.role === 'superadmin' ? 'Super Admin' : 'Batch Coordinator'}</span
								>
							</div>
						</div>

						<!-- Assigned Batch -->
						<div class="flex items-center gap-3">
							<svg
								xmlns="http://www.w3.org/2000/svg"
								fill="none"
								viewBox="0 0 24 24"
								stroke-width="2"
								stroke="currentColor"
								class="w-4 h-4 text-slate-400 shrink-0"
							>
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									d="M4.26 10.147a60.438 60.438 0 0 0-.491 6.347A48.62 48.62 0 0 1 12 20.9c4.956-1.9 8.219-4.787 8.219-4.787a60.43 60.43 0 0 0-.491-6.347M3.75 10.147 12 4.25l8.25 5.897m-16.5 0L12 16.023l8.25-5.876M8.25 10.147V16.5L12 19.5"
								/>
							</svg>
							<div class="flex items-center gap-1.5">
								<span class="text-slate-500 font-semibold">Assigned Batch:</span>
								<span class="font-bold text-slate-800">
									{formatBatch(admin.assigned_batch)}
								</span>
							</div>
						</div>
					</div>
				</div>
			</div>
		{/if}
	</div>

	<!-- Administrative Overview Row (DB-backed statistics only) -->
	{#if !loading && !error && admin}
		<div class="bg-white rounded-2xl border border-slate-200 shadow-sm p-5">
			<h2 class="font-serif font-bold text-sm text-slate-900">Administrative Overview</h2>
			<p class="text-xs text-slate-400 mt-0.5">Key metrics for your administrative activity</p>

			<!-- Stats Content -->
			<div class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-4 mt-4">
				<!-- Assigned Students -->
				<div
					class="rounded-xl border border-blue-200 bg-blue-50 p-4 hover:shadow-sm transition-shadow"
				>
					<p class="text-xs text-slate-600 leading-snug">Assigned Students</p>
					<p class="mt-3 text-3xl font-bold font-serif text-blue-600">
						{stats?.assigned_students ?? 0}
					</p>
				</div>

				<!-- Certificates Verified -->
				<div
					class="rounded-xl border border-emerald-200 bg-emerald-50 p-4 hover:shadow-sm transition-shadow"
				>
					<p class="text-xs text-slate-600 leading-snug">Certificates Verified</p>
					<p class="mt-3 text-3xl font-bold font-serif text-emerald-600">
						{stats?.verified_certificates ?? 0}
					</p>
				</div>

				<!-- Pending Reviews -->
				<div
					class="rounded-xl border border-amber-200 bg-amber-50 p-4 hover:shadow-sm transition-shadow"
				>
					<p class="text-xs text-slate-600 leading-snug">Pending Reviews</p>
					<p class="mt-3 text-3xl font-bold font-serif text-amber-600">
						{stats?.pending_reviews ?? 0}
					</p>
				</div>

				<!-- Activities Supervised -->
				<div
					class="rounded-xl border border-rose-200 bg-rose-50 p-4 hover:shadow-sm transition-shadow"
				>
					<p class="text-xs text-slate-600 leading-snug">Activities Supervised</p>
					<p class="mt-3 text-3xl font-bold font-serif text-[#881B1B]">
						{stats?.supervised_activities ?? 0}
					</p>
				</div>
			</div>
		</div>
	{/if}

	<!-- Assigned Batches Row -->
	{#if !loading && !error && admin}
		<!-- Assigned Batches Card -->
		<div class="bg-white rounded-2xl border border-slate-200 shadow-sm p-5">
			<h2 class="font-serif font-bold text-sm text-slate-900">Assigned Batches</h2>
			<p class="text-xs text-slate-400 mt-0.5">Batches currently under your coordination</p>

			<div class="flex-grow overflow-x-auto mt-4">
				<table class="w-full text-left text-xs border-collapse">
					<thead>
						<tr class="border-b border-slate-200">
							<th class="px-5 py-3 text-[11px] font-bold text-slate-400 uppercase tracking-wide"
								>Batch</th
							>
							<th class="px-5 py-3 text-[11px] font-bold text-slate-400 uppercase tracking-wide"
								>Course</th
							>
							<th class="px-5 py-3 text-[11px] font-bold text-slate-400 uppercase tracking-wide"
								>Semester</th
							>
							<th
								class="px-5 py-3 text-[11px] font-bold text-slate-400 uppercase tracking-wide text-right"
								>Students</th
							>
						</tr>
					</thead>
					<tbody>
						{#if statsLoading}
							<!-- Loading Rows -->
							{#each Array(2) as _}
								<tr class="animate-pulse border-b border-slate-100 last:border-b-0">
									<td class="px-5 py-3.5"><div class="h-4 bg-slate-200 rounded w-16"></div></td>
									<td class="px-5 py-3.5"><div class="h-4 bg-slate-200 rounded w-12"></div></td>
									<td class="px-5 py-3.5"><div class="h-4 bg-slate-200 rounded w-20"></div></td>
									<td class="px-5 py-3.5 text-right"
										><div class="h-4 bg-slate-200 rounded w-16 ml-auto"></div></td
									>
								</tr>
							{/each}
						{:else if statsError || assignedBatches.length === 0}
							<!-- Empty state placeholder -->
							<tr>
								<td colspan="4" class="px-5 py-12 text-center text-slate-400 font-medium font-sans">
									<svg
										xmlns="http://www.w3.org/2000/svg"
										fill="none"
										viewBox="0 0 24 24"
										stroke-width="1.5"
										stroke="currentColor"
										class="w-8 h-8 mx-auto mb-2 text-slate-300"
									>
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											d="M20.25 7.5l-.625 10.632a2.25 2.25 0 01-2.247 2.118H6.622a2.25 2.25 0 01-2.247-2.118L3.75 7.5M10 11.25h4M3.375 7.5h17.25c.621 0 1.125-.504 1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125H3.375c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125z"
										/>
									</svg>
									No assigned batches available
								</td>
							</tr>
						{:else}
							{#each assignedBatches as b}
								<tr class="border-b border-slate-100 last:border-b-0 hover:bg-slate-50/60">
									<td class="px-5 py-3.5 text-sm font-bold text-slate-900">{b.batch}</td>
									<td class="px-5 py-3.5 text-sm text-slate-700">{b.course}</td>
									<td class="px-5 py-3.5 text-sm text-slate-700">{b.semester}</td>
									<td class="px-5 py-3.5 text-right">
										<span
											class="px-2.5 py-1 bg-slate-100 text-slate-700 border border-slate-200 rounded-md text-[10px] font-bold"
										>
											{b.studentCount} Students
										</span>
									</td>
								</tr>
							{/each}
						{/if}
					</tbody>
				</table>
			</div>
		</div>
	{/if}
</div>

<!-- Change Password Modal Overlay -->
{#if isChangePasswordOpen}
	<div
		class="fixed inset-0 bg-slate-900/40 backdrop-blur-xs flex items-center justify-center p-4 z-50 animate-fade-in"
		transition:fade={{ duration: 100 }}
	>
		<div
			class="bg-white rounded-2xl border border-slate-200 shadow-xl max-w-md w-full p-6 space-y-4"
		>
			<div>
				<h3 class="text-lg font-bold text-slate-900 font-serif leading-tight">Change Password</h3>
				<p class="text-xs text-slate-500 font-medium mt-1">
					Update your account security credential password.
				</p>
			</div>

			<form onsubmit={handleChangePassword} class="space-y-4">
				{#if passwordError}
					<div
						class="p-3 text-xs font-semibold text-rose-650 bg-rose-50 border border-rose-100 rounded-lg"
					>
						{passwordError}
					</div>
				{/if}
				{#if passwordSuccess}
					<div
						class="p-3 text-xs font-semibold text-emerald-650 bg-emerald-50 border border-emerald-100 rounded-lg"
					>
						{passwordSuccess}
					</div>
				{/if}

				<div class="space-y-1.5">
					<label
						for="curr-password"
						class="text-[10px] font-bold text-slate-450 uppercase tracking-wider"
						>Current Password</label
					>
					<input
						id="curr-password"
						type="password"
						bind:value={currentPassword}
						disabled={isPasswordSubmitting}
						class="w-full px-3 py-2 border border-slate-200 focus:border-[#881B1B]/50 focus:ring-1 focus:ring-[#881B1B]/40 rounded-lg text-sm focus:outline-none bg-white disabled:bg-slate-50 disabled:text-slate-400 transition-colors"
					/>
				</div>

				<div class="space-y-1.5">
					<label
						for="new-password"
						class="text-[10px] font-bold text-slate-450 uppercase tracking-wider"
						>New Password</label
					>
					<input
						id="new-password"
						type="password"
						bind:value={newPassword}
						disabled={isPasswordSubmitting}
						class="w-full px-3 py-2 border border-slate-200 focus:border-[#881B1B]/50 focus:ring-1 focus:ring-[#881B1B]/40 rounded-lg text-sm focus:outline-none bg-white disabled:bg-slate-50 disabled:text-slate-400 transition-colors"
					/>
				</div>

				<div class="space-y-1.5">
					<label
						for="conf-password"
						class="text-[10px] font-bold text-slate-450 uppercase tracking-wider"
						>Confirm New Password</label
					>
					<input
						id="conf-password"
						type="password"
						bind:value={confirmPassword}
						disabled={isPasswordSubmitting}
						class="w-full px-3 py-2 border border-slate-200 focus:border-[#881B1B]/50 focus:ring-1 focus:ring-[#881B1B]/40 rounded-lg text-sm focus:outline-none bg-white disabled:bg-slate-50 disabled:text-slate-400 transition-colors"
					/>
				</div>

				<div class="flex items-center justify-end gap-3 pt-3 border-t border-slate-100">
					<button
						type="button"
						onclick={() => (isChangePasswordOpen = false)}
						disabled={isPasswordSubmitting}
						class="px-4 py-2 border border-slate-200 hover:bg-slate-50 disabled:opacity-50 text-slate-700 bg-white rounded-lg text-xs font-bold transition-colors cursor-pointer focus:outline-none"
					>
						Cancel
					</button>
					<button
						type="submit"
						disabled={isPasswordSubmitting}
						class="inline-flex items-center justify-center gap-1.5 px-4 py-2 bg-[#881B1B] hover:bg-[#881B1B]/90 disabled:bg-[#881B1B]/50 text-white rounded-lg text-xs font-bold transition-colors cursor-pointer focus:outline-none"
					>
						{#if isPasswordSubmitting}
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
							Changing...
						{:else}
							Change Password
						{/if}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}
