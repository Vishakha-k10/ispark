<script lang="ts">
	import { fade, slide } from 'svelte/transition';
	import { onMount } from 'svelte';
	import { API_BASE_URL } from '$lib/config';

	// ── Types ──────────────────────────────────────────────────────────────────
	type CertStatus = 'Pending' | 'Approved' | 'Rejected';

	interface Certificate {
		id: string;
		student: string;
		regNo: string;
		name: string;
		type: string;
		submittedOn: string;
		status: CertStatus;
		priority: boolean;
		relatedActivity: string;
		creditsRequested: number;
		remarks: string;
	}

	interface RawCert {
		id: number | string;
		student?: { name?: string };
		student_roll_no: string;
		activity_name: string;
		activity_category: string;
		created_at: string;
		status: CertStatus;
		credits: number;
		description?: string;
	}

	let certificates = $state<Certificate[]>([]);

	onMount(async () => {
		try {
			const token = localStorage.getItem('admin_token');
			if (!token) return;
			const res = await fetch(`${API_BASE_URL}/api/admin/certificates`, {
				headers: { Authorization: `Bearer ${token}` }
			});
			if (res.ok) {
				const data = await res.json();
				if (data.certificates) {
					certificates = data.certificates.map((c: RawCert) => ({
						id: String(c.id),
						student: c.student?.name || 'Unknown',
						regNo: c.student_roll_no,
						name: c.activity_name,
						type: c.activity_category,
						submittedOn: new Date(c.created_at).toLocaleDateString('en-GB', {
							day: '2-digit',
							month: 'short',
							year: 'numeric'
						}),
						status: c.status,
						priority: false,
						relatedActivity: c.activity_name,
						creditsRequested: c.credits,
						remarks: c.description || ''
					}));
				}
			}
		} catch (e) {
			console.error('Failed to load certificates', e);
		}
	});

	// ── Derived Stats ──────────────────────────────────────────────────────────
	const pendingCount = $derived(certificates.filter((c) => c.status === 'Pending').length);
	const approvedCount = $derived(certificates.filter((c) => c.status === 'Approved').length);
	const rejectedCount = $derived(certificates.filter((c) => c.status === 'Rejected').length);
	const verificationRate = $derived(
		approvedCount + rejectedCount === 0
			? 0
			: Math.round((approvedCount / (approvedCount + rejectedCount)) * 100)
	);

	// ── Queue: filters + pagination ──────────────────────────────────────────────
	type QueueFilter = 'All' | 'Pending' | 'Approved' | 'Rejected' | 'High Priority';
	const filterTabs: QueueFilter[] = ['All', 'Pending', 'Approved', 'Rejected', 'High Priority'];

	let activeFilter = $state<QueueFilter>('All');
	let searchQuery = $state('');
	let currentPage = $state(1);
	const pageSize = 8;

	const filteredCertificates = $derived(
		certificates.filter((c) => {
			const matchSearch =
				searchQuery === '' ||
				c.student.toLowerCase().includes(searchQuery.toLowerCase()) ||
				c.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
				c.id.toLowerCase().includes(searchQuery.toLowerCase()) ||
				c.type.toLowerCase().includes(searchQuery.toLowerCase());
			const matchFilter =
				activeFilter === 'All'
					? true
					: activeFilter === 'High Priority'
						? c.priority
						: c.status === activeFilter;
			return matchSearch && matchFilter;
		})
	);

	const totalPages = $derived(Math.max(1, Math.ceil(filteredCertificates.length / pageSize)));

	const pagedCertificates = $derived(
		filteredCertificates.slice((currentPage - 1) * pageSize, currentPage * pageSize)
	);

	function setFilter(f: QueueFilter) {
		activeFilter = f;
		currentPage = 1;
	}

	function resetPage() {
		currentPage = 1;
	}

	// ── Certificate Preview selection ────────────────────────────────────────────
	let selectedId = $state<string>('CERT-2041');
	const selectedCert = $derived(certificates.find((c) => c.id === selectedId) ?? certificates[0]);

	// ── Actions ──────────────────────────────────────────────────────────────────
	async function approveCert(cert: Certificate) {
		try {
			const token = localStorage.getItem('admin_token');
			const res = await fetch(`${API_BASE_URL}/api/admin/certificates/${cert.id}/approve`, {
				method: 'POST',
				headers: { Authorization: `Bearer ${token}` }
			});
			if (res.ok) {
				certificates = certificates.map((c) =>
					c.id === cert.id ? { ...c, status: 'Approved' as CertStatus } : c
				);
				triggerToast(`Approved “${cert.name}” for ${cert.student}.`);
			} else {
				triggerToast('Failed to approve certificate', 'danger');
			}
		} catch {
			triggerToast('Failed to approve certificate', 'danger');
		}
	}

	async function rejectCert(cert: Certificate) {
		const reason = prompt('Reason for rejection:');
		if (reason === null) return;
		try {
			const token = localStorage.getItem('admin_token');
			const res = await fetch(`${API_BASE_URL}/api/admin/certificates/${cert.id}/reject`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					Authorization: `Bearer ${token}`
				},
				body: JSON.stringify({ reason })
			});
			if (res.ok) {
				certificates = certificates.map((c) =>
					c.id === cert.id ? { ...c, status: 'Rejected' as CertStatus } : c
				);
				triggerToast(`Rejected “${cert.name}” for ${cert.student}.`, 'danger');
			} else {
				triggerToast('Failed to reject certificate', 'danger');
			}
		} catch {
			triggerToast('Failed to reject certificate', 'danger');
		}
	}

	const recentActivity = $derived(
		certificates
			.filter((c) => c.status !== 'Pending')
			.map((c) => ({
				student: c.student,
				regNo: c.regNo,
				certificate: c.name,
				date: c.submittedOn,
				verifiedBy: 'Admin',
				status: c.status
			}))
			.slice(0, 5)
	);

	// ── Toast ──────────────────────────────────────────────────────────────────
	interface Toast {
		id: number;
		message: string;
		type: 'success' | 'danger';
	}
	let toasts = $state<Toast[]>([]);
	let nextToastId = 0;

	function triggerToast(message: string, type: 'success' | 'danger' = 'success') {
		const id = nextToastId++;
		toasts = [...toasts, { id, message, type }];
		setTimeout(() => {
			toasts = toasts.filter((t) => t.id !== id);
		}, 3000);
	}

	// ── Helpers ──────────────────────────────────────────────────────────────────
	let isFullCertModalOpen = $state(false);

	function openFullCertModal(cert: Certificate) {
		selectedId = cert.id;
		isFullCertModalOpen = true;
	}

	function closeFullCertModal() {
		isFullCertModalOpen = false;
	}

	async function downloadCertificate(cert: Certificate) {
		try {
			const token = localStorage.getItem('admin_token');
			const res = await fetch(`${API_BASE_URL}/api/admin/certificates/${cert.id}/download`, {
				headers: token ? { Authorization: `Bearer ${token}` } : {}
			});

			if (res.ok) {
				const blob = await res.blob();
				const contentDisposition = res.headers.get('content-disposition');
				let filename = `Certificate_${cert.id}_${cert.regNo}`;
				if (contentDisposition) {
					const match = contentDisposition.match(/filename="?([^"]+)"?/);
					if (match && match[1]) {
						filename = match[1];
					}
				}
				const url = URL.createObjectURL(blob);
				const link = document.createElement('a');
				link.href = url;
				link.download = filename;
				document.body.appendChild(link);
				link.click();
				link.remove();
				URL.revokeObjectURL(url);
				triggerToast(`Downloaded submitted certificate for ${cert.student}`);
				return;
			}
			triggerToast('Certificate file is no longer available on server', 'danger');
		} catch (e) {
			console.error('Failed to download certificate file from backend', e);
			triggerToast('Failed to download certificate file from server', 'danger');
		}
	}

	function initials(name: string): string {
		return name
			.split(' ')
			.map((n) => n[0])
			.join('')
			.slice(0, 2)
			.toUpperCase();
	}

	function statusChip(status: CertStatus): string {
		switch (status) {
			case 'Approved':
				return 'bg-emerald-50 text-emerald-700 border border-emerald-100';
			case 'Pending':
				return 'bg-amber-50 text-amber-700 border border-amber-100';
			case 'Rejected':
				return 'bg-rose-50 text-rose-700 border border-rose-100';
		}
	}

	function statusDot(status: CertStatus): string {
		switch (status) {
			case 'Approved':
				return 'bg-emerald-500';
			case 'Pending':
				return 'bg-amber-500';
			case 'Rejected':
				return 'bg-rose-500';
		}
	}

	function typeChip(type: string): string {
		switch (type) {
			case 'Competition':
				return 'bg-purple-50 text-purple-700 border border-purple-100';
			case 'Course':
				return 'bg-blue-50 text-blue-700 border border-blue-100';
			case 'Seminar':
				return 'bg-amber-50 text-amber-700 border border-amber-100';
			case 'Sports':
				return 'bg-emerald-50 text-emerald-700 border border-emerald-100';
			case 'Cultural':
				return 'bg-rose-50 text-rose-700 border border-rose-100';
			case 'Research':
				return 'bg-indigo-50 text-indigo-700 border border-indigo-100';
			case 'Service':
				return 'bg-teal-50 text-teal-700 border border-teal-100';
			case 'Workshop':
				return 'bg-orange-50 text-orange-700 border border-orange-100';
			default:
				return 'bg-slate-100 text-slate-600 border border-slate-200';
		}
	}
</script>

<!-- ── Toast Container ─────────────────────────────────────────────────────── -->
<div class="fixed bottom-4 right-4 z-50 flex flex-col gap-2 max-w-sm pointer-events-none">
	{#each toasts as toast (toast.id)}
		<div
			transition:slide={{ duration: 150 }}
			class="p-4 rounded-xl border shadow-lg flex items-center gap-3 text-xs font-semibold font-sans pointer-events-auto {toast.type ===
			'success'
				? 'bg-emerald-50 border-emerald-200 text-emerald-800'
				: 'bg-rose-50 border-rose-200 text-rose-800'}"
		>
			{#if toast.type === 'success'}
				<svg
					xmlns="http://www.w3.org/2000/svg"
					fill="none"
					viewBox="0 0 24 24"
					stroke-width="2.5"
					stroke="currentColor"
					class="w-4 h-4 text-emerald-600 shrink-0"
				>
					<path stroke-linecap="round" stroke-linejoin="round" d="m4.5 12.75 6 6 9-13.5" />
				</svg>
			{:else}
				<svg
					xmlns="http://www.w3.org/2000/svg"
					fill="none"
					viewBox="0 0 24 24"
					stroke-width="2.5"
					stroke="currentColor"
					class="w-4 h-4 text-rose-600 shrink-0"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M12 9v3.75m9-.75a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-9 3.75h.008v.008H12v-.008Z"
					/>
				</svg>
			{/if}
			<span>{toast.message}</span>
		</div>
	{/each}
</div>

<!-- ── Stat Cards ──────────────────────────────────────────────────────────── -->
<section class="grid grid-cols-2 lg:grid-cols-4 gap-4">
	<!-- Pending Verification -->
	<div
		class="bg-white p-5 rounded-xl border border-slate-200 flex flex-col justify-between shadow-xs hover:shadow-md transition-shadow duration-200"
	>
		<div class="flex items-center justify-between">
			<span class="text-2xl font-bold font-serif text-slate-900">{pendingCount}</span>
			<div class="p-2.5 rounded-lg bg-amber-50 text-amber-600 border border-amber-100">
				<svg
					xmlns="http://www.w3.org/2000/svg"
					fill="none"
					viewBox="0 0 24 24"
					stroke-width="2"
					stroke="currentColor"
					class="w-5 h-5"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M12 6v6h4.5m4.5 0a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z"
					/>
				</svg>
			</div>
		</div>
		<div class="mt-4">
			<h3 class="text-xs font-bold text-slate-800 tracking-wide">Pending Verification</h3>
			<p class="text-[10px] font-bold text-amber-500 mt-1 uppercase tracking-wider">
				Awaiting review
			</p>
		</div>
	</div>

	<!-- Approved This Month -->
	<div
		class="bg-white p-5 rounded-xl border border-slate-200 flex flex-col justify-between shadow-xs hover:shadow-md transition-shadow duration-200"
	>
		<div class="flex items-center justify-between">
			<span class="text-2xl font-bold font-serif text-slate-900">{approvedCount}</span>
			<div class="p-2.5 rounded-lg bg-emerald-50 text-emerald-600 border border-emerald-100">
				<svg
					xmlns="http://www.w3.org/2000/svg"
					fill="none"
					viewBox="0 0 24 24"
					stroke-width="2"
					stroke="currentColor"
					class="w-5 h-5"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M9 12.75 11.25 15 15 9.75M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z"
					/>
				</svg>
			</div>
		</div>
		<div class="mt-4">
			<h3 class="text-xs font-bold text-slate-800 tracking-wide">Approved This Month</h3>
			<p class="text-[10px] font-bold text-emerald-500 mt-1 uppercase tracking-wider">
				+8 this week
			</p>
		</div>
	</div>

	<!-- Rejected Certificates -->
	<div
		class="bg-white p-5 rounded-xl border border-slate-200 flex flex-col justify-between shadow-xs hover:shadow-md transition-shadow duration-200"
	>
		<div class="flex items-center justify-between">
			<span class="text-2xl font-bold font-serif text-slate-900">{rejectedCount}</span>
			<div class="p-2.5 rounded-lg bg-rose-50 text-rose-600 border border-rose-100">
				<svg
					xmlns="http://www.w3.org/2000/svg"
					fill="none"
					viewBox="0 0 24 24"
					stroke-width="2"
					stroke="currentColor"
					class="w-5 h-5"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M9.75 9.75l4.5 4.5m0-4.5l-4.5 4.5M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z"
					/>
				</svg>
			</div>
		</div>
		<div class="mt-4">
			<h3 class="text-xs font-bold text-slate-800 tracking-wide">Rejected Certificates</h3>
			<p class="text-[10px] font-bold text-rose-500 mt-1 uppercase tracking-wider">
				Requires resubmission
			</p>
		</div>
	</div>

	<!-- Verification Rate -->
	<div
		class="bg-white p-5 rounded-xl border border-slate-200 flex flex-col justify-between shadow-xs hover:shadow-md transition-shadow duration-200"
	>
		<div class="flex items-center justify-between">
			<span class="text-2xl font-bold font-serif text-slate-900">{verificationRate}%</span>
			<div class="p-2.5 rounded-lg bg-purple-50 text-purple-600 border border-purple-100">
				<svg
					xmlns="http://www.w3.org/2000/svg"
					fill="none"
					viewBox="0 0 24 24"
					stroke-width="2"
					stroke="currentColor"
					class="w-5 h-5"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M2.25 18 9 11.25l4.306 4.306a11.95 11.95 0 0 1 5.814-5.518l2.74-1.22m0 0-5.94-2.281m5.94 2.28-2.28 5.941"
					/>
				</svg>
			</div>
		</div>
		<div class="mt-4">
			<h3 class="text-xs font-bold text-slate-800 tracking-wide">Verification Rate</h3>
			<p class="text-[10px] font-bold text-slate-400 mt-1 uppercase tracking-wider">
				Approval accuracy
			</p>
		</div>
	</div>
</section>

<!-- ── Certificate Verification Queue ──────────────────────────────────────── -->
<section class="bg-white border border-slate-200 rounded-xl shadow-xs overflow-hidden">
	<!-- Queue header -->
	<div
		class="p-5 border-b border-slate-100 flex flex-col sm:flex-row sm:items-center justify-between gap-3 bg-slate-50/20"
	>
		<div class="flex items-center gap-2.5">
			<h2 class="text-sm font-bold font-serif text-inst-navy">Certificate Verification Queue</h2>
			<span
				class="text-[10px] font-extrabold text-amber-700 bg-amber-50 border border-amber-100 px-2 py-0.5 rounded-md uppercase tracking-wide"
				>{pendingCount} Pending</span
			>
		</div>
		<span class="text-[10px] font-extrabold text-slate-400 uppercase tracking-wider">
			{filteredCertificates.length} of {certificates.length} certificates
		</span>
	</div>

	<!-- Filter tabs + search -->
	<div
		class="px-5 py-3.5 border-b border-slate-100 flex flex-wrap items-center justify-between gap-3"
	>
		<div class="flex flex-wrap items-center gap-1.5">
			{#each filterTabs as tab}
				<button
					onclick={() => setFilter(tab)}
					class="px-3 py-1.5 text-[11px] font-bold rounded-lg border transition-colors {activeFilter ===
					tab
						? 'bg-[#881B1B] text-white border-[#881B1B]'
						: 'text-slate-500 border-slate-200 hover:bg-slate-50'}"
				>
					{tab}
				</button>
			{/each}
		</div>

		<div class="relative flex items-center">
			<span class="absolute left-3 text-slate-400 pointer-events-none">
				<svg
					xmlns="http://www.w3.org/2000/svg"
					fill="none"
					viewBox="0 0 24 24"
					stroke-width="2"
					stroke="currentColor"
					class="w-3.5 h-3.5"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
					/>
				</svg>
			</span>
			<input
				type="text"
				placeholder="Search student or certificate..."
				bind:value={searchQuery}
				oninput={resetPage}
				class="pl-8 pr-4 py-2 bg-slate-50 rounded-lg border border-slate-200 text-xs text-slate-800 focus:outline-none focus:border-slate-350 focus:bg-white w-56 transition-all focus:w-64"
			/>
		</div>
	</div>

	<!-- Queue table -->
	<div class="overflow-x-auto">
		<table class="w-full text-left border-collapse">
			<thead>
				<tr
					class="border-b border-slate-100 bg-slate-50/50 text-[10px] font-extrabold text-slate-400 uppercase tracking-wider"
				>
					<th class="py-3 px-5">Cert ID</th>
					<th class="py-3 px-5">Student</th>
					<th class="py-3 px-5">Certificate Name</th>
					<th class="py-3 px-5">Type</th>
					<th class="py-3 px-5">Submitted On</th>
					<th class="py-3 px-5">Status</th>
					<th class="py-3 px-5 text-center">Actions</th>
				</tr>
			</thead>
			<tbody class="divide-y divide-slate-100 text-xs font-sans">
				{#if pagedCertificates.length === 0}
					<tr>
						<td colspan="7" class="py-16 text-center text-slate-400 font-semibold text-xs">
							No certificates match your filters.
						</td>
					</tr>
				{:else}
					{#each pagedCertificates as cert (cert.id)}
						<tr
							onclick={() => (selectedId = cert.id)}
							class="cursor-pointer transition-colors {selectedId === cert.id
								? 'bg-[#881B1B]/5'
								: 'hover:bg-slate-50/50'}"
						>
							<!-- Cert ID -->
							<td class="py-3.5 px-5">
								<span class="font-extrabold text-slate-400 text-[10px] uppercase tracking-wider"
									>{cert.id}</span
								>
							</td>
							<!-- Student -->
							<td class="py-3.5 px-5">
								<div class="flex items-center gap-2">
									{#if cert.priority}
										<span
											class="w-1.5 h-1.5 rounded-full bg-[#881B1B] shrink-0"
											title="High priority"
										></span>
									{/if}
									<div>
										<div class="font-bold text-slate-800">{cert.student}</div>
										<div class="text-[10px] text-slate-400 font-semibold uppercase">
											{cert.regNo}
										</div>
									</div>
								</div>
							</td>
							<!-- Certificate Name -->
							<td class="py-3.5 px-5 font-bold text-slate-700">{cert.name}</td>
							<!-- Type -->
							<td class="py-3.5 px-5">
								<span
									class="inline-flex px-2 py-0.5 rounded-md text-[10px] font-bold {typeChip(
										cert.type
									)}">{cert.type}</span
								>
							</td>
							<!-- Submitted On -->
							<td class="py-3.5 px-5 text-slate-500 font-semibold">{cert.submittedOn}</td>
							<!-- Status -->
							<td class="py-3.5 px-5">
								<span
									class="inline-flex items-center gap-1.5 px-2 py-1 rounded-md text-[10px] font-extrabold uppercase tracking-wide {statusChip(
										cert.status
									)}"
								>
									<span class="w-1.5 h-1.5 rounded-full {statusDot(cert.status)}"></span>
									{cert.status}
								</span>
							</td>
							<!-- Actions -->
							<td class="py-3.5 px-5">
								<div class="flex items-center justify-center gap-2">
									<button
										onclick={(e) => {
											e.stopPropagation();
											approveCert(cert);
										}}
										disabled={cert.status === 'Approved'}
										class="inline-flex items-center gap-1 px-2.5 py-1.5 text-[10px] font-bold text-emerald-600 border border-emerald-200 bg-emerald-50 rounded-lg hover:bg-emerald-100 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
									>
										<svg
											xmlns="http://www.w3.org/2000/svg"
											fill="none"
											viewBox="0 0 24 24"
											stroke-width="2.5"
											stroke="currentColor"
											class="w-3 h-3"
										>
											<path
												stroke-linecap="round"
												stroke-linejoin="round"
												d="m4.5 12.75 6 6 9-13.5"
											/>
										</svg>
										Approve
									</button>
									<button
										onclick={(e) => {
											e.stopPropagation();
											rejectCert(cert);
										}}
										disabled={cert.status === 'Rejected'}
										class="inline-flex items-center gap-1 px-2.5 py-1.5 text-[10px] font-bold text-rose-600 border border-rose-200 bg-rose-50 rounded-lg hover:bg-rose-100 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
									>
										<svg
											xmlns="http://www.w3.org/2000/svg"
											fill="none"
											viewBox="0 0 24 24"
											stroke-width="2.5"
											stroke="currentColor"
											class="w-3 h-3"
										>
											<path
												stroke-linecap="round"
												stroke-linejoin="round"
												d="M6 18 18 6M6 6l12 12"
											/>
										</svg>
										Reject
									</button>
								</div>
							</td>
						</tr>
					{/each}
				{/if}
			</tbody>
		</table>
	</div>

	<!-- Queue footer: info + pagination -->
	<div
		class="px-5 py-3.5 border-t border-slate-100 flex flex-col sm:flex-row items-center justify-between gap-3 bg-slate-50/20"
	>
		<span class="text-[11px] font-bold text-slate-400 uppercase tracking-wider">
			Showing {filteredCertificates.length === 0
				? 0
				: Math.min((currentPage - 1) * pageSize + 1, filteredCertificates.length)}–{Math.min(
				currentPage * pageSize,
				filteredCertificates.length
			)} of {filteredCertificates.length} certificates
		</span>

		<div class="flex items-center gap-1.5">
			<button
				onclick={() => {
					if (currentPage > 1) currentPage -= 1;
				}}
				disabled={currentPage === 1}
				class="w-8 h-8 flex items-center justify-center rounded-lg border border-slate-200 text-slate-500 hover:bg-slate-100 disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
				aria-label="Previous page"
			>
				<svg
					xmlns="http://www.w3.org/2000/svg"
					fill="none"
					viewBox="0 0 24 24"
					stroke-width="2.5"
					stroke="currentColor"
					class="w-3.5 h-3.5"
				>
					<path stroke-linecap="round" stroke-linejoin="round" d="M15.75 19.5 8.25 12l7.5-7.5" />
				</svg>
			</button>

			{#each Array.from({ length: totalPages }, (_, i) => i + 1) as page}
				<button
					onclick={() => (currentPage = page)}
					class="w-8 h-8 flex items-center justify-center rounded-lg border text-xs font-extrabold transition-colors {currentPage ===
					page
						? 'bg-[#881B1B] text-white border-[#881B1B]'
						: 'border-slate-200 text-slate-600 hover:bg-slate-100'}"
				>
					{page}
				</button>
			{/each}

			<button
				onclick={() => {
					if (currentPage < totalPages) currentPage += 1;
				}}
				disabled={currentPage === totalPages}
				class="w-8 h-8 flex items-center justify-center rounded-lg border border-slate-200 text-slate-500 hover:bg-slate-100 disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
				aria-label="Next page"
			>
				<svg
					xmlns="http://www.w3.org/2000/svg"
					fill="none"
					viewBox="0 0 24 24"
					stroke-width="2.5"
					stroke="currentColor"
					class="w-3.5 h-3.5"
				>
					<path stroke-linecap="round" stroke-linejoin="round" d="m8.25 4.5 7.5 7.5-7.5 7.5" />
				</svg>
			</button>
		</div>
	</div>
</section>

<!-- ── Certificate Preview ─────────────────────────────────────────────────── -->
<section class="bg-white border border-slate-200 rounded-xl shadow-xs overflow-hidden">
	<div
		class="p-5 border-b border-slate-100 flex flex-col sm:flex-row sm:items-center justify-between gap-2 bg-slate-50/20"
	>
		<h2 class="text-sm font-bold font-serif text-inst-navy">Certificate Preview</h2>
		<span class="text-[10px] font-bold text-slate-400 uppercase tracking-wider">
			Click any row above to load the certificate
		</span>
	</div>

	{#if selectedCert}
		<div class="p-5 grid grid-cols-1 lg:grid-cols-12 gap-5">
			<!-- Image placeholder -->
			<div class="lg:col-span-4 flex flex-col gap-3">
				<div
					class="aspect-4/3 rounded-xl border-2 border-dashed border-slate-200 bg-slate-50 flex flex-col items-center justify-center gap-2 text-slate-400"
				>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						fill="none"
						viewBox="0 0 24 24"
						stroke-width="1.5"
						stroke="currentColor"
						class="w-10 h-10"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="m2.25 15.75 5.159-5.159a2.25 2.25 0 0 1 3.182 0l5.159 5.159m-1.5-1.5 1.409-1.409a2.25 2.25 0 0 1 3.182 0l2.909 2.909m-18 3.75h16.5a1.5 1.5 0 0 0 1.5-1.5V6a1.5 1.5 0 0 0-1.5-1.5H3.75A1.5 1.5 0 0 0 2.25 6v12a1.5 1.5 0 0 0 1.5 1.5Zm10.5-11.25h.008v.008h-.008V8.25Zm.375 0a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Z"
						/>
					</svg>
					<span class="text-[11px] font-bold">Certificate Image</span>
					<span class="text-[10px] font-semibold text-slate-400">{selectedCert.student}</span>
				</div>
				<button
					onclick={() => openFullCertModal(selectedCert!)}
					class="w-full inline-flex items-center justify-center gap-1.5 py-2 text-[11px] font-bold text-slate-600 border border-slate-200 rounded-lg hover:bg-slate-50 transition-colors cursor-pointer"
				>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						fill="none"
						viewBox="0 0 24 24"
						stroke-width="2"
						stroke="currentColor"
						class="w-3.5 h-3.5"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M2.036 12.322a1.012 1.012 0 0 1 0-.639C3.423 7.51 7.36 4.5 12 4.5c4.638 0 8.573 3.007 9.963 7.178.07.207.07.431 0 .639C20.577 16.49 16.64 19.5 12 19.5c-4.638 0-8.573-3.007-9.963-7.178Z"
						/>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M15 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z"
						/>
					</svg>
					View Full Certificate
				</button>
			</div>

			<!-- Details -->
			<div class="lg:col-span-8 flex flex-col gap-4">
				<div>
					<div class="flex items-center gap-2.5 flex-wrap">
						<h3 class="text-lg font-bold font-serif text-slate-900">{selectedCert.name}</h3>
						<span
							class="inline-flex items-center gap-1.5 px-2 py-1 rounded-md text-[10px] font-extrabold uppercase tracking-wide {statusChip(
								selectedCert.status
							)}"
						>
							<span class="w-1.5 h-1.5 rounded-full {statusDot(selectedCert.status)}"></span>
							{selectedCert.status}
						</span>
					</div>
					<p class="text-[11px] font-semibold text-slate-400 mt-0.5">
						Submitted by {selectedCert.student} · {selectedCert.regNo} · {selectedCert.submittedOn}
					</p>
				</div>

				<!-- Info grid -->
				<div
					class="grid grid-cols-2 sm:grid-cols-3 gap-4 bg-slate-50 p-4 rounded-xl border border-slate-150"
				>
					<div>
						<span class="text-[9px] font-bold text-slate-400 uppercase tracking-wider block"
							>Related Activity</span
						>
						<span class="text-xs font-bold text-slate-800 block mt-0.5"
							>{selectedCert.relatedActivity}</span
						>
					</div>
					<div>
						<span class="text-[9px] font-bold text-slate-400 uppercase tracking-wider block"
							>Certificate Type</span
						>
						<span class="text-xs font-bold text-slate-800 block mt-0.5">{selectedCert.type}</span>
					</div>
					<div>
						<span class="text-[9px] font-bold text-slate-400 uppercase tracking-wider block"
							>Credits Requested</span
						>
						<span class="text-xs font-bold text-[#881B1B] block mt-0.5"
							>{selectedCert.creditsRequested} credits</span
						>
					</div>
					<div>
						<span class="text-[9px] font-bold text-slate-400 uppercase tracking-wider block"
							>Submission Date</span
						>
						<span class="text-xs font-bold text-slate-800 block mt-0.5"
							>{selectedCert.submittedOn}</span
						>
					</div>
					<div>
						<span class="text-[9px] font-bold text-slate-400 uppercase tracking-wider block"
							>Status</span
						>
						<span class="text-xs font-bold text-slate-800 block mt-0.5">{selectedCert.status}</span>
					</div>
					<div>
						<span class="text-[9px] font-bold text-slate-400 uppercase tracking-wider block"
							>Cert ID</span
						>
						<span class="text-xs font-bold text-slate-800 block mt-0.5">{selectedCert.id}</span>
					</div>
				</div>

				<!-- Remarks -->
				<div class="rounded-xl border border-slate-150 p-4">
					<span class="text-[9px] font-bold text-slate-400 uppercase tracking-wider block mb-1"
						>Student Remarks</span
					>
					<p class="text-xs text-slate-600 font-medium leading-relaxed">{selectedCert.remarks}</p>
				</div>

				<!-- Actions -->
				<div class="flex flex-wrap items-center gap-3">
					<button
						onclick={() => approveCert(selectedCert)}
						disabled={selectedCert.status === 'Approved'}
						class="inline-flex items-center gap-1.5 px-4 py-2 text-xs font-bold text-white bg-emerald-600 hover:bg-emerald-700 rounded-lg transition-colors shadow-xs disabled:opacity-40 disabled:cursor-not-allowed"
					>
						<svg
							xmlns="http://www.w3.org/2000/svg"
							fill="none"
							viewBox="0 0 24 24"
							stroke-width="2.5"
							stroke="currentColor"
							class="w-3.5 h-3.5"
						>
							<path stroke-linecap="round" stroke-linejoin="round" d="m4.5 12.75 6 6 9-13.5" />
						</svg>
						Approve Certificate
					</button>
					<button
						onclick={() => rejectCert(selectedCert)}
						disabled={selectedCert.status === 'Rejected'}
						class="inline-flex items-center gap-1.5 px-4 py-2 text-xs font-bold text-white bg-rose-600 hover:bg-rose-700 rounded-lg transition-colors shadow-xs disabled:opacity-40 disabled:cursor-not-allowed"
					>
						<svg
							xmlns="http://www.w3.org/2000/svg"
							fill="none"
							viewBox="0 0 24 24"
							stroke-width="2.5"
							stroke="currentColor"
							class="w-3.5 h-3.5"
						>
							<path stroke-linecap="round" stroke-linejoin="round" d="M6 18 18 6M6 6l12 12" />
						</svg>
						Reject Certificate
					</button>
					<button
						onclick={() => downloadCertificate(selectedCert!)}
						class="inline-flex items-center gap-1.5 px-4 py-2 text-xs font-bold text-slate-600 border border-slate-200 rounded-lg hover:bg-slate-50 transition-colors cursor-pointer"
					>
						<svg
							xmlns="http://www.w3.org/2000/svg"
							fill="none"
							viewBox="0 0 24 24"
							stroke-width="2"
							stroke="currentColor"
							class="w-3.5 h-3.5"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								d="M3 16.5v2.25A2.25 2.25 0 0 0 5.25 21h13.5A2.25 2.25 0 0 0 21 18.75V16.5m-13.5-9L12 3m0 0 4.5 4.5M12 3v13.5"
							/>
						</svg>
						Download
					</button>
				</div>
			</div>
		</div>
	{/if}
</section>

<!-- ── Recent Verification Activity ────────────────────────────────────────── -->
<section class="bg-white border border-slate-200 rounded-xl shadow-xs overflow-hidden">
	<div class="p-5 border-b border-slate-100 bg-slate-50/20">
		<h2 class="text-sm font-bold font-serif text-inst-navy">Recent Verification Activity</h2>
	</div>
	<div class="overflow-x-auto">
		<table class="w-full text-left border-collapse">
			<thead>
				<tr
					class="border-b border-slate-100 bg-slate-50/50 text-[10px] font-extrabold text-slate-400 uppercase tracking-wider"
				>
					<th class="py-3 px-5">Student</th>
					<th class="py-3 px-5">Reg No.</th>
					<th class="py-3 px-5">Certificate</th>
					<th class="py-3 px-5">Verification Date</th>
					<th class="py-3 px-5">Verified By</th>
					<th class="py-3 px-5">Status</th>
				</tr>
			</thead>
			<tbody class="divide-y divide-slate-100 text-xs font-sans">
				{#each recentActivity as log}
					<tr class="hover:bg-slate-50/50 transition-colors">
						<td class="py-3.5 px-5">
							<div class="flex items-center gap-2.5">
								<div
									class="w-7 h-7 rounded-full bg-[#881B1B]/10 text-[#881B1B] flex items-center justify-center font-bold text-[10px] font-serif shrink-0"
								>
									{initials(log.student)}
								</div>
								<span class="font-bold text-slate-800">{log.student}</span>
							</div>
						</td>
						<td class="py-3.5 px-5 text-slate-500 font-semibold uppercase text-[11px]"
							>{log.regNo}</td
						>
						<td class="py-3.5 px-5 text-slate-700 font-semibold">{log.certificate}</td>
						<td class="py-3.5 px-5 text-slate-500 font-semibold">{log.date}</td>
						<td class="py-3.5 px-5 text-slate-600 font-semibold">{log.verifiedBy}</td>
						<td class="py-3.5 px-5">
							<span
								class="inline-flex items-center gap-1.5 px-2 py-1 rounded-md text-[10px] font-extrabold uppercase tracking-wide {statusChip(
									log.status
								)}"
							>
								<span class="w-1.5 h-1.5 rounded-full {statusDot(log.status)}"></span>
								{log.status}
							</span>
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
</section>

<!-- ── Full Certificate Modal ───────────────────────────────────────────── -->
{#if isFullCertModalOpen && selectedCert}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/50 backdrop-blur-xs p-4"
		transition:fade={{ duration: 150 }}
	>
		<div
			class="w-full max-w-lg rounded-2xl border border-slate-200 bg-white p-6 shadow-xl"
			transition:slide={{ duration: 150 }}
		>
			<div class="flex items-center justify-between border-b border-slate-100 pb-4">
				<div>
					<span class="text-[10px] font-extrabold tracking-wider text-slate-400 uppercase"
						>Certificate Verification Record</span
					>
					<h3 class="text-base font-bold text-slate-900">{selectedCert.name}</h3>
				</div>
				<button
					aria-label="Close"
					onclick={closeFullCertModal}
					class="rounded-lg p-1 text-slate-400 hover:bg-slate-100 hover:text-slate-600 cursor-pointer"
				>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						class="h-5 w-5"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M6 18L18 6M6 6l12 12"
						/>
					</svg>
				</button>
			</div>
			<div class="mt-4 space-y-3 font-sans text-xs">
				<div class="flex justify-between rounded-lg bg-slate-50 p-3">
					<span class="font-bold text-slate-500">Certificate ID</span>
					<span class="font-extrabold text-slate-800">{selectedCert.id}</span>
				</div>
				<div class="flex justify-between rounded-lg bg-slate-50 p-3">
					<span class="font-bold text-slate-500">Student Name</span>
					<span class="font-extrabold text-slate-800">{selectedCert.student}</span>
				</div>
				<div class="flex justify-between rounded-lg bg-slate-50 p-3">
					<span class="font-bold text-slate-500">Roll No</span>
					<span class="font-bold text-slate-800">{selectedCert.regNo}</span>
				</div>
				<div class="flex justify-between rounded-lg bg-slate-50 p-3">
					<span class="font-bold text-slate-500">Certificate Type</span>
					<span class="font-bold text-slate-800">{selectedCert.type}</span>
				</div>
				<div class="flex justify-between rounded-lg bg-slate-50 p-3">
					<span class="font-bold text-slate-500">Submitted Date</span>
					<span class="font-bold text-slate-800">{selectedCert.submittedOn}</span>
				</div>
				<div class="flex justify-between rounded-lg bg-slate-50 p-3">
					<span class="font-bold text-slate-500">Credits Requested</span>
					<span class="font-extrabold text-emerald-700"
						>{selectedCert.creditsRequested} Credits</span
					>
				</div>
				<div class="flex justify-between rounded-lg bg-slate-50 p-3">
					<span class="font-bold text-slate-500">Verification Status</span>
					<span class="font-bold text-slate-800">{selectedCert.status}</span>
				</div>
				{#if selectedCert.remarks}
					<div class="rounded-lg bg-slate-50 p-3">
						<span class="font-bold text-slate-500 block mb-1">Remarks / Description</span>
						<p class="font-medium text-slate-700">{selectedCert.remarks}</p>
					</div>
				{/if}
			</div>
			<div class="mt-6 flex justify-end gap-2">
				<button
					onclick={() => downloadCertificate(selectedCert!)}
					class="inline-flex items-center gap-1.5 rounded-lg border border-slate-200 px-4 py-2 text-xs font-bold text-slate-700 hover:bg-slate-50 cursor-pointer"
				>
					Download Record
				</button>
				<button
					onclick={closeFullCertModal}
					class="rounded-lg bg-[#881B1B] px-4 py-2 text-xs font-bold text-white hover:bg-[#881B1B]/90 cursor-pointer"
				>
					Close
				</button>
			</div>
		</div>
	</div>
{/if}
