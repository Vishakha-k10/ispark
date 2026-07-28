<script lang="ts">
	import { fade, slide } from 'svelte/transition';
	import { onMount } from 'svelte';
	import { API_BASE_URL } from '$lib/config';
	import AdminStudentDetailView from './AdminStudentDetailView.svelte';

	// Optional prop to filter students by batch when navigated from Batch Analytics
	// In Svelte 5 runes mode use $props() instead of `export let`
	const props = $props<{ batch?: string }>();

	// ── Types ──────────────────────────────────────────────────────────────────
	type Status = 'Active' | 'At Risk' | 'Pending Review' | 'Inactive';

	interface Student {
		id: string;
		name: string;
		regNo: string;
		department: string;
		semester: number;
		creditsEarned: number;
		creditsTarget: number;
		certificates: number;
		pendingCertificates: number;
		activityCount: number;
		status: Status;
		email: string;
		batch: string;
	}

	type HistoryStatus = 'Completed' | 'Pending' | 'Rejected';

	interface BackendCertificate {
		id: number;
		activity_name: string;
		organizer_name: string;
		activity_date: string;
		credits: number;
		status: string;
	}

	interface BackendEnrollment {
		id: number;
		status: string;
		activity?: {
			name: string;
			category: string;
			activity_date: string;
			credits: number;
		};
	}

	interface BackendStudent {
		roll_no?: string;
		name: string;
		course_name?: string;
		semester?: number;
		email_id?: string;
		credits_earned?: number;
		activity_count?: number;
		pending_certificates?: number;
		engagement_status?: string;
		total_certificates?: number;
		certificates?: BackendCertificate[];
		enrollments?: BackendEnrollment[];
	}

	function formatDate(value: string | undefined): string {
		if (!value) return '—';
		const parsed = new Date(value);
		return Number.isNaN(parsed.getTime())
			? '—'
			: parsed.toLocaleDateString('en-GB', { day: '2-digit', month: 'short', year: 'numeric' });
	}

	function toHistoryStatus(status: string): HistoryStatus {
		if (status === 'Approved' || status === 'Completed') return 'Completed';
		if (status === 'Rejected' || status === 'Cancelled') return 'Rejected';
		return 'Pending';
	}

	const STATUSES: Status[] = ['Active', 'At Risk', 'Pending Review', 'Inactive'];

	function toStatus(value: string | undefined): Status {
		return STATUSES.includes(value as Status) ? (value as Status) : 'Active';
	}

	// ── State Variables ────────────────────────────────────────────────────────
	let allStudents = $state<Student[]>([]);

	function mapBackendStudent(s: BackendStudent): Student {
		return {
			id: s.roll_no ?? '',
			name: s.name,
			regNo: s.roll_no ?? '',
			department: s.course_name ?? '',
			semester: s.semester ?? 0,
			creditsEarned: s.credits_earned ?? 0,
			creditsTarget: 200,
			certificates: s.total_certificates ?? 0,
			pendingCertificates: s.pending_certificates ?? 0,
			activityCount: s.activity_count ?? 0,
			status: toStatus(s.engagement_status),
			email: s.email_id ?? '',
			batch: s.roll_no?.match(/^[A-Z]+2K\d+/)?.[0] ?? ''
		};
	}

	// ── Data Fetching ──────────────────────────────────────────────────────────
	onMount(async () => {
		try {
			const token = localStorage.getItem('admin_token');
			const response = await fetch(`${API_BASE_URL}/api/admin/students`, {
				headers: { Authorization: `Bearer ${token}` }
			});

			if (!response.ok) throw new Error('Failed to fetch students');

			const data = await response.json();

			allStudents = data.students.map(mapBackendStudent);
		} catch {
			triggerToast('Failed to load student data', 'danger');
		}
	});

	// ── Derived Stats ──────────────────────────────────────────────────────────
	const totalStudents = $derived(allStudents.length);
	const activeStudents = $derived(allStudents.filter((s) => s.status === 'Active').length);
	const pendingCertReviews = $derived(
		allStudents.reduce((sum, s) => sum + s.pendingCertificates, 0)
	);
	const avgCredits = $derived(
		allStudents.length > 0
			? Math.round(allStudents.reduce((sum, s) => sum + s.creditsEarned, 0) / allStudents.length)
			: 0
	);

	// Student Overview highlights
	const perfScore = (s: Student) => s.creditsEarned + s.certificates * 15 + s.activityCount * 2;
	const topPerformer = $derived(
		allStudents.length > 0 ? [...allStudents].sort((a, b) => perfScore(b) - perfScore(a))[0] : null
	);
	const highestCredits = $derived(
		allStudents.length > 0
			? [...allStudents].sort((a, b) => b.creditsEarned - a.creditsEarned)[0]
			: null
	);
	const mostActive = $derived(
		allStudents.length > 0
			? [...allStudents].sort((a, b) => b.activityCount - a.activityCount)[0]
			: null
	);
	const pendingAttention = $derived(
		allStudents.filter((s) => s.status === 'At Risk' || s.status === 'Pending Review')
	);

	// ── Table State ────────────────────────────────────────────────────────────
	let searchQuery = $state('');
	let filterStatus = $state<Status | 'All'>('All');
	let filterDept = $state('All');
	let currentPage = $state(1);
	const pageSize = 10;
	let showFilters = $state(false);

	// Must be derived so it updates after the fetch completes!
	const departments = $derived([
		'All',
		...Array.from(new Set(allStudents.map((s) => s.department)))
	]);

	const filteredStudents = $derived(
		allStudents.filter((s) => {
			const matchSearch =
				searchQuery === '' ||
				s.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
				s.regNo.toLowerCase().includes(searchQuery.toLowerCase()) ||
				s.department.toLowerCase().includes(searchQuery.toLowerCase());
			const matchStatus = filterStatus === 'All' || s.status === filterStatus;
			const matchDept = filterDept === 'All' || s.department === filterDept;
			const incoming = props.batch ?? '';
			const matchBatch = incoming === '' || s.batch === incoming;
			return matchSearch && matchStatus && matchDept && matchBatch;
		})
	);

	const totalPages = $derived(Math.max(1, Math.ceil(filteredStudents.length / pageSize)));
	const pagedStudents = $derived(
		filteredStudents.slice((currentPage - 1) * pageSize, currentPage * pageSize)
	);

	function resetPage() {
		currentPage = 1;
	}

	// ── Modal ──────────────────────────────────────────────────────────────────
	interface HistoryActivity {
		id: number;
		name: string;
		category: string;
		date: string;
		credits: number;
		status: HistoryStatus;
	}

	interface HistoryCertificate {
		id: number;
		name: string;
		issuer: string;
		date: string;
		credits: number;
		status: HistoryStatus;
	}

	let activeStudent = $state<Student | null>(null);
	let detailActivities = $state<HistoryActivity[]>([]);
	let detailCertificates = $state<HistoryCertificate[]>([]);
	let isModalOpen = $state(false);

	async function openStudentModal(student: Student) {
		try {
			const token = localStorage.getItem('admin_token');
			const res = await fetch(`${API_BASE_URL}/api/admin/students/${student.regNo}`, {
				headers: { Authorization: `Bearer ${token}` }
			});

			if (res.ok) {
				const data = await res.json();
				const detail: BackendStudent = data.student;
				activeStudent = mapBackendStudent(detail);
				detailCertificates = (detail.certificates ?? []).map((cert) => ({
					id: cert.id,
					name: cert.activity_name,
					issuer: cert.organizer_name || '—',
					date: formatDate(cert.activity_date),
					credits: cert.credits,
					status: toHistoryStatus(cert.status)
				}));

				detailActivities = (detail.enrollments ?? [])
					.filter((enrollment) => enrollment.activity)
					.map((enrollment) => ({
						id: enrollment.id,
						name: enrollment.activity!.name,
						category: enrollment.activity!.category,
						date: formatDate(enrollment.activity!.activity_date),
						credits: enrollment.activity!.credits,
						status: toHistoryStatus(enrollment.status)
					}));
			} else {
				activeStudent = student;
			}
		} catch {
			activeStudent = student;
		}
		isModalOpen = true;
	}

	function closeModal() {
		isModalOpen = false;
		activeStudent = null;
	}

	async function handleSendNotice() {
		if (!activeStudent) return;
		const message = prompt(`Enter notice message for ${activeStudent.name}:`);
		if (!message || message.trim() === '') return;

		triggerToast(`Sending notice to ${activeStudent.name}...`);
		const token = localStorage.getItem('admin_token');
		try {
			const res = await fetch(`${API_BASE_URL}/api/admin/students/${activeStudent.regNo}/notice`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					Authorization: `Bearer ${token}`
				},
				body: JSON.stringify({ message })
			});
			if (res.ok) {
				triggerToast(`Notice sent successfully!`);
			} else {
				triggerToast(`Failed to send notice`, 'danger');
			}
		} catch {
			triggerToast(`Failed to send notice`, 'danger');
		}
		closeModal();
	}

	// ── Student Detail View ──────────────────────────────────────────────────────
	let detailStudent = $state<Student | null>(null);

	function studentRank(student: Student): number {
		return allStudents.filter((s) => s.creditsEarned > student.creditsEarned).length + 1;
	}

	function openStudentDetail(student: Student) {
		detailStudent = student;
		isModalOpen = false;
		activeStudent = null;
	}

	function closeStudentDetail() {
		detailStudent = null;
	}

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

	// ── Helpers ────────────────────────────────────────────────────────────────
	function statusClass(status: Status): string {
		switch (status) {
			case 'Active':
				return 'bg-emerald-50 text-emerald-700 border border-emerald-100';
			case 'At Risk':
				return 'bg-rose-50 text-rose-700 border border-rose-100';
			case 'Pending Review':
				return 'bg-amber-50 text-amber-700 border border-amber-100';
			case 'Inactive':
				return 'bg-slate-100 text-slate-500 border border-slate-200';
		}
	}

	function statusDot(status: Status): string {
		switch (status) {
			case 'Active':
				return 'bg-emerald-500';
			case 'At Risk':
				return 'bg-rose-500';
			case 'Pending Review':
				return 'bg-amber-500';
			case 'Inactive':
				return 'bg-slate-400';
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
</script>

<!-- ── Toast Container ─────────────────────────────────────────────────────── -->
<div class="pointer-events-none fixed right-4 bottom-4 z-50 flex max-w-sm flex-col gap-2">
	{#each toasts as toast (toast.id)}
		<div
			transition:slide={{ duration: 150 }}
			class="pointer-events-auto flex items-center gap-3 rounded-xl border p-4 font-sans text-xs font-semibold shadow-lg {toast.type ===
			'success'
				? 'border-emerald-200 bg-emerald-50 text-emerald-800'
				: 'border-rose-200 bg-rose-50 text-rose-800'}"
		>
			{#if toast.type === 'success'}
				<svg
					xmlns="http://www.w3.org/2000/svg"
					fill="none"
					viewBox="0 0 24 24"
					stroke-width="2.5"
					stroke="currentColor"
					class="h-4 w-4 shrink-0 text-emerald-600"
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
					class="h-4 w-4 shrink-0 text-rose-600"
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

{#if detailStudent}
	<AdminStudentDetailView
		student={detailStudent}
		rank={studentRank(detailStudent)}
		cohortSize={totalStudents}
		activities={detailActivities}
		certificates={detailCertificates}
		onBack={closeStudentDetail}
		onToast={triggerToast}
	/>
{:else}
	<!-- ── Stat Cards ──────────────────────────────────────────────────────────── -->
	<section class="grid grid-cols-2 gap-4 lg:grid-cols-4">
		<!-- Total Students -->
		<div
			class="flex flex-col justify-between rounded-xl border border-slate-200 bg-white p-5 shadow-xs transition-shadow duration-200 hover:shadow-md"
		>
			<div class="flex items-center justify-between">
				<span class="font-serif text-2xl font-bold text-slate-900">{totalStudents}</span>
				<div class="rounded-lg border border-blue-100 bg-blue-50 p-2.5 text-blue-600">
					<svg
						xmlns="http://www.w3.org/2000/svg"
						fill="none"
						viewBox="0 0 24 24"
						stroke-width="2"
						stroke="currentColor"
						class="h-5 w-5"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M18 18.72a9.094 9.094 0 0 0 3.741-.479 3 3 0 0 0-4.682-2.72m.94 3.198.001.031c0 .225-.012.447-.037.666A11.944 11.944 0 0 1 12 21c-2.17 0-4.207-.576-5.963-1.584A6.062 6.062 0 0 1 6 18.719m12 0a5.971 5.971 0 0 0-.941-3.197m0 0A5.995 5.995 0 0 0 12 12.75a5.995 5.995 0 0 0-5.058 2.772m0 0a3 3 0 0 0-4.681 2.72 8.986 8.986 0 0 0 3.74.477m.94-3.197a5.971 5.971 0 0 0-.94 3.197M15 6.75a3 3 0 1 1-6 0 3 3 0 0 1 6 0Zm6 3a2.25 2.25 0 1 1-4.5 0 2.25 2.25 0 0 1 4.5 0Zm-13.5 0a2.25 2.25 0 1 1-4.5 0 2.25 2.25 0 0 1 4.5 0Z"
						/>
					</svg>
				</div>
			</div>
			<div class="mt-4">
				<h3 class="text-xs font-bold tracking-wide text-slate-800">Total Students</h3>
				<p class="mt-1 text-[10px] font-bold tracking-wider text-slate-400 uppercase">
					+2 this semester
				</p>
			</div>
		</div>

		<!-- Active Students -->
		<div
			class="flex flex-col justify-between rounded-xl border border-slate-200 bg-white p-5 shadow-xs transition-shadow duration-200 hover:shadow-md"
		>
			<div class="flex items-center justify-between">
				<span class="font-serif text-2xl font-bold text-slate-900">{activeStudents}</span>
				<div class="rounded-lg border border-emerald-100 bg-emerald-50 p-2.5 text-emerald-600">
					<svg
						xmlns="http://www.w3.org/2000/svg"
						fill="none"
						viewBox="0 0 24 24"
						stroke-width="2"
						stroke="currentColor"
						class="h-5 w-5"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M15.75 6a3.75 3.75 0 1 1-7.5 0 3.75 3.75 0 0 1 7.5 0ZM4.501 20.118a7.5 7.5 0 0 1 14.998 0A17.933 17.933 0 0 1 12 21.75c-2.676 0-5.216-.584-7.499-1.632Z"
						/>
					</svg>
				</div>
			</div>
			<div class="mt-4">
				<h3 class="text-xs font-bold tracking-wide text-slate-800">Active Students</h3>
				<p class="mt-1 text-[10px] font-bold tracking-wider text-slate-400 uppercase">
					{activeStudents} on engagement
				</p>
			</div>
		</div>

		<!-- Pending Certificate Reviews -->
		<div
			class="flex flex-col justify-between rounded-xl border border-slate-200 bg-white p-5 shadow-xs transition-shadow duration-200 hover:shadow-md"
		>
			<div class="flex items-center justify-between">
				<span class="font-serif text-2xl font-bold text-slate-900">{pendingCertReviews}</span>
				<div class="rounded-lg border border-amber-100 bg-amber-50 p-2.5 text-amber-600">
					<svg
						xmlns="http://www.w3.org/2000/svg"
						fill="none"
						viewBox="0 0 24 24"
						stroke-width="2"
						stroke="currentColor"
						class="h-5 w-5"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M19.5 14.25v-2.625a3.375 3.375 0 0 0-3.375-3.375h-1.5A1.125 1.125 0 0 1 13.5 7.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 0 0-9-9Z"
						/>
					</svg>
				</div>
			</div>
			<div class="mt-4">
				<h3 class="text-xs font-bold tracking-wide text-slate-800">Pending Certificate Reviews</h3>
				<p class="mt-1 text-[10px] font-bold tracking-wider text-slate-400 uppercase">
					3 marked urgent
				</p>
			</div>
		</div>

		<!-- Average Credits Earned -->
		<div
			class="flex flex-col justify-between rounded-xl border border-slate-200 bg-white p-5 shadow-xs transition-shadow duration-200 hover:shadow-md"
		>
			<div class="flex items-center justify-between">
				<span class="font-serif text-2xl font-bold text-slate-900">{avgCredits}</span>
				<div class="rounded-lg border border-purple-100 bg-purple-50 p-2.5 text-purple-600">
					<svg
						xmlns="http://www.w3.org/2000/svg"
						fill="none"
						viewBox="0 0 24 24"
						stroke-width="2"
						stroke="currentColor"
						class="h-5 w-5"
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
				<h3 class="text-xs font-bold tracking-wide text-slate-800">Average Credits Earned</h3>
				<p class="mt-1 text-[10px] font-bold tracking-wider text-slate-400 uppercase">
					57.5% avg from last batch
				</p>
			</div>
		</div>
	</section>

	<!-- ── Student Overview + Quick Insights ──────────────────────────────────── -->
	<section class="grid grid-cols-1 items-stretch gap-5 lg:grid-cols-12">
		<!-- Student Overview -->
		<div
			class="flex flex-col overflow-hidden rounded-xl border border-slate-200 bg-white shadow-xs lg:col-span-8"
		>
			<div class="border-b border-slate-100 bg-slate-50/20 p-5">
				<h2 class="text-inst-navy font-serif text-sm font-bold">Student Overview</h2>
			</div>
			<div class="grid flex-grow auto-rows-fr grid-cols-1 gap-4 p-5 sm:grid-cols-2">
				<!-- Top Performer -->
				<div
					class="flex items-center gap-4 rounded-xl border border-slate-300/70 bg-white p-5 shadow-2xs transition-shadow hover:shadow-sm"
				>
					<div
						class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-yellow-200 bg-amber-50 text-amber-500"
					>
						<svg
							xmlns="http://www.w3.org/2000/svg"
							fill="none"
							viewBox="0 0 24 24"
							stroke-width="2"
							stroke="currentColor"
							class="h-5 w-5"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								d="M11.48 3.499a.562.562 0 0 1 1.04 0l2.125 5.111a.563.563 0 0 0 .475.345l5.518.442c.499.04.701.663.321.988l-4.204 3.602a.563.563 0 0 0-.182.557l1.285 5.385a.562.562 0 0 1-.84.61l-4.725-2.885a.562.562 0 0 0-.586 0L6.982 20.54a.562.562 0 0 1-.84-.61l1.285-5.386a.562.562 0 0 0-.182-.557l-4.204-3.602a.562.562 0 0 1 .321-.988l5.518-.442a.563.563 0 0 0 .475-.345L11.48 3.5Z"
							/>
						</svg>
					</div>
					<div class="flex min-w-0 flex-col gap-0.5">
						<span class="text-[10px] font-bold tracking-wider text-slate-400 uppercase"
							>TOP PERFORMER</span
						>
						<span class="truncate text-base font-bold text-slate-900">
							{topPerformer?.name || 'Waiting for data...'}
						</span>
						<span class="text-xs font-semibold text-slate-500">
							{topPerformer?.creditsEarned || 0} credits · {topPerformer?.department || '...'}
						</span>
					</div>
				</div>

				<!-- Highest Credits Earned -->
				<div
					class="flex items-center gap-4 rounded-xl border border-slate-300/70 bg-white p-5 shadow-2xs transition-shadow hover:shadow-sm"
				>
					<div
						class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-emerald-200 bg-emerald-50 text-emerald-500"
					>
						<svg
							xmlns="http://www.w3.org/2000/svg"
							fill="none"
							viewBox="0 0 24 24"
							stroke-width="2"
							stroke="currentColor"
							class="h-5 w-5"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								d="M2.25 18 9 11.25l4.306 4.306a11.95 11.95 0 0 1 5.814-5.518l2.74-1.22m0 0-5.94-2.281m5.94 2.28-2.28 5.941"
							/>
						</svg>
					</div>
					<div class="flex min-w-0 flex-col gap-0.5">
						<span class="text-[10px] font-bold tracking-wider text-slate-400 uppercase"
							>HIGHEST CREDITS EARNED</span
						>
						<span class="truncate text-base font-bold text-slate-900">
							{highestCredits?.name || 'Waiting for data...'}
						</span>
						<span class="text-xs font-semibold text-slate-500">
							{highestCredits?.creditsEarned || 0} credits earned this batch
						</span>
					</div>
				</div>

				<!-- Most Active Student -->
				<div
					class="flex items-center gap-4 rounded-xl border border-slate-300/70 bg-white p-5 shadow-2xs transition-shadow hover:shadow-sm"
				>
					<div
						class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-blue-200 bg-blue-50 text-blue-500"
					>
						<svg
							xmlns="http://www.w3.org/2000/svg"
							fill="none"
							viewBox="0 0 24 24"
							stroke-width="2"
							stroke="currentColor"
							class="h-5 w-5"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								d="M3.75 13.5l10.5-11.25L12 10.5h8.25L9.75 21.75 12 13.5H3.75z"
							/>
						</svg>
					</div>
					<div class="flex min-w-0 flex-col gap-0.5">
						<span class="text-[10px] font-bold tracking-wider text-slate-400 uppercase"
							>MOST ACTIVE STUDENT</span
						>
						<span class="truncate text-base font-bold text-slate-900">
							{mostActive?.name || 'Waiting for data...'}
						</span>
						<span class="text-xs font-semibold text-slate-500">
							{mostActive?.activityCount || 0} activities logged
						</span>
					</div>
				</div>

				<!-- Requiring Attention -->
				<div
					class="flex items-center gap-4 rounded-xl border border-rose-200/80 bg-rose-50/50 p-5 shadow-2xs transition-shadow hover:shadow-sm"
				>
					<div
						class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-rose-200 bg-rose-100 text-rose-600"
					>
						<svg
							xmlns="http://www.w3.org/2000/svg"
							fill="none"
							viewBox="0 0 24 24"
							stroke-width="2"
							stroke="currentColor"
							class="h-5 w-5"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								d="M12 9v3.75m9-.75a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-9 3.75h.008v.008H12v-.008Z"
							/>
						</svg>
					</div>
					<div class="flex min-w-0 flex-col gap-0.5">
						<span class="text-[10px] font-bold tracking-wider text-rose-500 uppercase"
							>REQUIRING ATTENTION</span
						>
						<span class="text-base font-bold text-rose-700">{pendingAttention.length} students</span
						>
						<span class="text-xs font-semibold text-rose-500">
							{pendingAttention.length > 0
								? pendingAttention
										.slice(0, 4)
										.map((s) => s.name.split(' ')[0])
										.join(', ')
								: 'None'}
						</span>
					</div>
				</div>
			</div>
		</div>

		<!-- Quick Insights -->
		<div
			class="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-xs lg:col-span-4"
		>
			<div class="border-b border-slate-100 bg-slate-50/20 p-5">
				<h2 class="text-inst-navy font-serif text-sm font-bold">Quick Insights</h2>
			</div>
			<div class="space-y-3 p-4">
				<!-- Pending certificates -->
				<div
					class="flex cursor-pointer items-center justify-between rounded-lg border border-slate-100 p-3 transition-colors hover:bg-slate-50"
				>
					<div class="flex items-center gap-2.5">
						<div
							class="flex h-7 w-7 items-center justify-center rounded-lg border border-amber-100 bg-amber-50 text-amber-600"
						>
							<svg
								xmlns="http://www.w3.org/2000/svg"
								fill="none"
								viewBox="0 0 24 24"
								stroke-width="2"
								stroke="currentColor"
								class="h-3.5 w-3.5"
							>
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									d="M19.5 14.25v-2.625a3.375 3.375 0 0 0-3.375-3.375h-1.5A1.125 1.125 0 0 1 13.5 7.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H8.25m6.75 12H9m1.5-12H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 0 0-9-9Z"
								/>
							</svg>
						</div>
						<div>
							<p class="text-[11px] font-bold text-slate-700">Pending certificates</p>
							<p class="text-[9px] font-semibold text-slate-400">Need review</p>
						</div>
					</div>
					<span
						class="rounded-md border border-amber-100 bg-amber-50 px-2 py-0.5 text-xs font-extrabold text-amber-600"
						>{pendingCertReviews}</span
					>
				</div>

				<!-- Review credit target -->
				<div
					class="flex cursor-pointer items-center justify-between rounded-lg border border-slate-100 p-3 transition-colors hover:bg-slate-50"
				>
					<div class="flex items-center gap-2.5">
						<div
							class="flex h-7 w-7 items-center justify-center rounded-lg border border-blue-100 bg-blue-50 text-blue-600"
						>
							<svg
								xmlns="http://www.w3.org/2000/svg"
								fill="none"
								viewBox="0 0 24 24"
								stroke-width="2"
								stroke="currentColor"
								class="h-3.5 w-3.5"
							>
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									d="M2.25 18 9 11.25l4.306 4.306a11.95 11.95 0 0 1 5.814-5.518l2.74-1.22m0 0-5.94-2.281m5.94 2.28-2.28 5.941"
								/>
							</svg>
						</div>
						<div>
							<p class="text-[11px] font-bold text-slate-700">Review credit target</p>
							<p class="text-[9px] font-semibold text-slate-400">Below threshold</p>
						</div>
					</div>
					<span
						class="rounded-md border border-blue-100 bg-blue-50 px-2 py-0.5 text-xs font-extrabold text-blue-600"
						>4</span
					>
				</div>

				<!-- Inactive students -->
				<div
					class="flex cursor-pointer items-center justify-between rounded-lg border border-slate-100 p-3 transition-colors hover:bg-slate-50"
				>
					<div class="flex items-center gap-2.5">
						<div
							class="flex h-7 w-7 items-center justify-center rounded-lg border border-slate-200 bg-slate-100 text-slate-500"
						>
							<svg
								xmlns="http://www.w3.org/2000/svg"
								fill="none"
								viewBox="0 0 24 24"
								stroke-width="2"
								stroke="currentColor"
								class="h-3.5 w-3.5"
							>
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									d="M15.75 6a3.75 3.75 0 1 1-7.5 0 3.75 3.75 0 0 1 7.5 0ZM4.501 20.118a7.5 7.5 0 0 1 14.998 0A17.933 17.933 0 0 1 12 21.75c-2.676 0-5.216-.584-7.499-1.632Z"
								/>
							</svg>
						</div>
						<div>
							<p class="text-[11px] font-bold text-slate-700">Inactive students</p>
							<p class="text-[9px] font-semibold text-slate-400">30 days</p>
						</div>
					</div>
					<span
						class="rounded-md border border-slate-200 bg-slate-100 px-2 py-0.5 text-xs font-extrabold text-slate-600"
						>3</span
					>
				</div>

				<!-- Pending review -->
				<div
					class="flex cursor-pointer items-center justify-between rounded-lg border border-slate-100 p-3 transition-colors hover:bg-slate-50"
				>
					<div class="flex items-center gap-2.5">
						<div
							class="flex h-7 w-7 items-center justify-center rounded-lg border border-rose-100 bg-rose-50 text-rose-600"
						>
							<svg
								xmlns="http://www.w3.org/2000/svg"
								fill="none"
								viewBox="0 0 24 24"
								stroke-width="2"
								stroke="currentColor"
								class="h-3.5 w-3.5"
							>
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									d="M12 9v3.75m9-.75a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-9 3.75h.008v.008H12v-.008Z"
								/>
							</svg>
						</div>
						<div>
							<p class="text-[11px] font-bold text-slate-700">Pending review</p>
							<p class="text-[9px] font-semibold text-slate-400">Action required</p>
						</div>
					</div>
					<span
						class="rounded-md border border-rose-100 bg-rose-50 px-2 py-0.5 text-xs font-extrabold text-rose-600"
						>{pendingAttention.length}</span
					>
				</div>

				<div class="pt-1">
					<button
						onclick={() => {
							filterStatus = 'At Risk';
							resetPage();
						}}
						class="w-full rounded-lg border border-[#881B1B]/20 bg-[#881B1B]/5 py-2 text-[11px] font-bold tracking-wide text-[#881B1B] uppercase transition-colors hover:bg-[#881B1B]/10"
					>
						View All Flagged Students
					</button>
				</div>
			</div>
		</div>
	</section>

	<!-- ── Student Management Table ───────────────────────────────────────────── -->
	<section class="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-xs">
		<!-- Table Header -->
		<div
			class="flex flex-col justify-between gap-3 border-b border-slate-100 bg-slate-50/20 p-5 sm:flex-row sm:items-center"
		>
			<h2 class="text-inst-navy font-serif text-sm font-bold">Student Management</h2>
			<div
				class="flex items-center gap-2 text-[10px] font-extrabold tracking-wider text-slate-400 uppercase"
			>
				Total Students: <span class="ml-1 text-slate-700">{filteredStudents.length}</span>
			</div>
		</div>

		<!-- Search & Filter Bar -->
		<div class="flex flex-wrap items-center gap-3 border-b border-slate-100 px-5 py-3.5">
			<!-- Search -->
			<div class="relative flex items-center">
				<span class="pointer-events-none absolute left-3 text-slate-400">
					<svg
						xmlns="http://www.w3.org/2000/svg"
						fill="none"
						viewBox="0 0 24 24"
						stroke-width="2"
						stroke="currentColor"
						class="h-3.5 w-3.5"
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
					placeholder="Search students..."
					bind:value={searchQuery}
					oninput={resetPage}
					class="focus:border-slate-350 w-48 rounded-lg border border-slate-200 bg-slate-50 py-2 pr-4 pl-8 text-xs text-slate-800 transition-all focus:w-56 focus:bg-white focus:outline-none"
				/>
			</div>

			<!-- More Filters toggle -->
			<button
				onclick={() => (showFilters = !showFilters)}
				class="flex items-center gap-1.5 rounded-lg border border-slate-200 px-3 py-2 text-[11px] font-bold text-slate-500 transition-colors hover:bg-slate-50"
			>
				<svg
					xmlns="http://www.w3.org/2000/svg"
					fill="none"
					viewBox="0 0 24 24"
					stroke-width="2"
					stroke="currentColor"
					class="h-3.5 w-3.5"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M10.5 6h9.75M10.5 6a1.5 1.5 0 1 1-3 0m3 0a1.5 1.5 0 1 0-3 0M3.75 6H7.5m3 12h9.75m-9.75 0a1.5 1.5 0 0 1-3 0m3 0a1.5 1.5 0 0 0-3 0m-3.75 0H7.5m9-6h3.75m-3.75 0a1.5 1.5 0 0 1-3 0m3 0a1.5 1.5 0 0 0-3 0m-9.75 0h9.75"
					/>
				</svg>
				More Filters
			</button>

			{#if showFilters}
				<div transition:slide={{ duration: 150 }} class="flex flex-wrap items-center gap-2">
					<select
						bind:value={filterStatus}
						onchange={resetPage}
						class="focus:border-slate-350 rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-[11px] font-bold text-slate-600 focus:outline-none"
					>
						<option value="All">All Status</option>
						<option value="Active">Active</option>
						<option value="At Risk">At Risk</option>
						<option value="Pending Review">Pending Review</option>
						<option value="Inactive">Inactive</option>
					</select>

					<select
						bind:value={filterDept}
						onchange={resetPage}
						class="focus:border-slate-350 rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-[11px] font-bold text-slate-600 focus:outline-none"
					>
						{#each departments as dept}
							<option value={dept}>{dept === 'All' ? 'All Departments' : dept}</option>
						{/each}
					</select>

					{#if filterStatus !== 'All' || filterDept !== 'All' || searchQuery !== ''}
						<button
							onclick={() => {
								filterStatus = 'All';
								filterDept = 'All';
								searchQuery = '';
								resetPage();
							}}
							class="rounded-lg border border-rose-200 bg-rose-50 px-3 py-2 text-[11px] font-bold text-rose-600 transition-colors hover:bg-rose-100"
						>
							Clear
						</button>
					{/if}
				</div>
			{/if}
		</div>

		<!-- Table -->
		<div class="overflow-x-auto">
			<table class="w-full border-collapse text-left">
				<thead>
					<tr
						class="border-b border-slate-100 bg-slate-50/50 text-[10px] font-extrabold tracking-wider text-slate-400 uppercase"
					>
						<th class="px-5 py-3">Name</th>
						<th class="px-5 py-3">Department</th>
						<th class="px-5 py-3">Sem</th>
						<th class="px-5 py-3">Credits Earned</th>
						<th class="px-5 py-3">Certificates</th>
						<th class="px-5 py-3">Activity Count</th>
						<th class="px-5 py-3">Status</th>
						<th class="px-5 py-3 text-center">Actions</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-slate-100 font-sans text-xs">
					{#if pagedStudents.length === 0}
						<tr>
							<td colspan="8" class="py-16 text-center text-xs font-semibold text-slate-400">
								No students match your search criteria.
							</td>
						</tr>
					{:else}
						{#each pagedStudents as student (student.id)}
							<tr class="transition-colors hover:bg-slate-50/50">
								<!-- Name -->
								<td class="px-5 py-3.5">
									<div class="font-bold text-slate-800">{student.name}</div>
									<div class="text-[10px] font-semibold text-slate-400 uppercase">
										{student.regNo}
									</div>
								</td>
								<!-- Department -->
								<td class="px-5 py-3.5 font-semibold text-slate-600">{student.department}</td>
								<!-- Semester -->
								<td class="px-5 py-3.5 font-extrabold text-slate-600">{student.semester}th</td>
								<!-- Credits Earned -->
								<td class="px-5 py-3.5">
									<div class="flex items-baseline gap-1">
										<span class="font-extrabold text-slate-800">{student.creditsEarned}</span>
										<span class="text-[9px] font-bold text-slate-400">/{student.creditsTarget}</span
										>
									</div>
									<div class="mt-1.5 h-1 w-20 overflow-hidden rounded-full bg-slate-100">
										<div
											class="h-full rounded-full {student.creditsEarned >= 150
												? 'bg-emerald-400'
												: student.creditsEarned >= 100
													? 'bg-amber-400'
													: 'bg-rose-400'}"
											style="width: {Math.min(
												100,
												(student.creditsEarned / student.creditsTarget) * 100
											)}%"
										></div>
									</div>
								</td>
								<!-- Certificates -->
								<td class="px-5 py-3.5">
									<span class="font-extrabold text-[#881B1B]">{student.certificates}</span>
								</td>
								<!-- Activity Count -->
								<td class="px-5 py-3.5 font-extrabold text-slate-700">{student.activityCount}</td>
								<!-- Status -->
								<td class="px-5 py-3.5">
									<span
										class="inline-flex items-center gap-1.5 rounded-md px-2 py-1 text-[10px] font-extrabold tracking-wide uppercase {statusClass(
											student.status
										)}"
									>
										<span class="h-1.5 w-1.5 rounded-full {statusDot(student.status)}"></span>
										{student.status}
									</span>
								</td>
								<!-- Actions -->
								<td class="px-5 py-3.5">
									<div class="flex items-center justify-center">
										<button
											onclick={() => openStudentModal(student)}
											aria-label="View student"
											class="flex h-8 w-8 items-center justify-center rounded-2xl bg-[#881B1B]/10 text-[#881B1B] transition-colors hover:bg-[#881B1B]/20 focus:outline-none"
										>
											<svg
												xmlns="http://www.w3.org/2000/svg"
												fill="none"
												viewBox="0 0 24 24"
												stroke-width="2"
												stroke="currentColor"
												class="h-4 w-4"
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
										</button>
									</div>
								</td>
							</tr>
						{/each}
					{/if}
				</tbody>
			</table>
		</div>

		<!-- Table Footer: info + pagination -->
		<div
			class="flex flex-col items-center justify-between gap-3 border-t border-slate-100 bg-slate-50/20 px-5 py-3.5 sm:flex-row"
		>
			<span class="text-[11px] font-bold tracking-wider text-slate-400 uppercase">
				Showing {filteredStudents.length === 0
					? 0
					: Math.min((currentPage - 1) * pageSize + 1, filteredStudents.length)}–{Math.min(
					currentPage * pageSize,
					filteredStudents.length
				)} of {filteredStudents.length} students
			</span>

			<div class="flex items-center gap-1.5">
				<button
					onclick={() => {
						if (currentPage > 1) currentPage -= 1;
					}}
					disabled={currentPage === 1}
					class="flex h-8 w-8 items-center justify-center rounded-lg border border-slate-200 text-slate-500 transition-colors hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-30"
					aria-label="Previous page"
				>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						fill="none"
						viewBox="0 0 24 24"
						stroke-width="2.5"
						stroke="currentColor"
						class="h-3.5 w-3.5"
					>
						<path stroke-linecap="round" stroke-linejoin="round" d="M15.75 19.5 8.25 12l7.5-7.5" />
					</svg>
				</button>

				{#each Array.from({ length: totalPages }, (_, i) => i + 1) as page}
					<button
						onclick={() => (currentPage = page)}
						class="flex h-8 w-8 items-center justify-center rounded-lg border text-xs font-extrabold transition-colors {currentPage ===
						page
							? 'border-[#881B1B] bg-[#881B1B] text-white'
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
					class="flex h-8 w-8 items-center justify-center rounded-lg border border-slate-200 text-slate-500 transition-colors hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-30"
					aria-label="Next page"
				>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						fill="none"
						viewBox="0 0 24 24"
						stroke-width="2.5"
						stroke="currentColor"
						class="h-3.5 w-3.5"
					>
						<path stroke-linecap="round" stroke-linejoin="round" d="m8.25 4.5 7.5 7.5-7.5 7.5" />
					</svg>
				</button>
			</div>
		</div>
	</section>
{/if}

<!-- ── Student Detail Modal ───────────────────────────────────────────────── -->
{#if isModalOpen && activeStudent}
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div
		transition:fade={{ duration: 150 }}
		class="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/60 p-4 backdrop-blur-xs"
		onclick={closeModal}
	>
		<div
			onclick={(e) => e.stopPropagation()}
			class="flex w-full max-w-lg flex-col overflow-hidden rounded-2xl border border-slate-200 bg-white font-sans shadow-2xl"
		>
			<!-- Modal Header -->
			<div class="border-slate-150 flex items-center justify-between border-b bg-slate-50/30 p-5">
				<div>
					<h3 class="text-inst-navy font-serif text-sm font-bold">Student Profile</h3>
					<p class="mt-0.5 text-[9px] font-bold tracking-widest text-slate-400 uppercase">
						ID: {activeStudent.id} · Batch {activeStudent.batch}
					</p>
				</div>
				<button
					onclick={closeModal}
					aria-label="Close modal"
					class="rounded-lg p-1 text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-600"
				>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						fill="none"
						viewBox="0 0 24 24"
						stroke-width="2"
						stroke="currentColor"
						class="h-5 w-5"
					>
						<path stroke-linecap="round" stroke-linejoin="round" d="M6 18 18 6M6 6l12 12" />
					</svg>
				</button>
			</div>

			<!-- Modal Body -->
			<div class="max-h-[65vh] space-y-5 overflow-y-auto p-6">
				<!-- Avatar + Name -->
				<div class="flex items-center gap-4">
					<div
						class="flex h-14 w-14 shrink-0 items-center justify-center rounded-full border-2 border-white bg-[#881B1B] font-serif text-lg font-bold text-white shadow-md"
					>
						{initials(activeStudent.name)}
					</div>
					<div class="flex-grow">
						<div class="font-serif text-lg font-bold text-slate-900">{activeStudent.name}</div>
						<div class="text-[10px] font-bold tracking-wider text-slate-400 uppercase">
							{activeStudent.regNo}
						</div>
						<div class="text-[10px] font-semibold text-slate-500">{activeStudent.email}</div>
					</div>
					<span
						class="inline-flex items-center gap-1.5 rounded-lg px-2.5 py-1.5 text-[10px] font-extrabold tracking-wide uppercase {statusClass(
							activeStudent.status
						)}"
					>
						<span class="h-1.5 w-1.5 rounded-full {statusDot(activeStudent.status)}"></span>
						{activeStudent.status}
					</span>
				</div>

				<!-- Info Grid -->
				<div class="border-slate-150 grid grid-cols-2 gap-4 rounded-xl border bg-slate-50 p-4">
					<div>
						<span class="block text-[9px] font-bold tracking-wider text-slate-400 uppercase"
							>Department</span
						>
						<span class="mt-0.5 block text-xs font-bold text-slate-800"
							>{activeStudent.department}</span
						>
					</div>
					<div>
						<span class="block text-[9px] font-bold tracking-wider text-slate-400 uppercase"
							>Semester</span
						>
						<span class="mt-0.5 block text-xs font-bold text-slate-800"
							>{activeStudent.semester}th Semester</span
						>
					</div>
					<div>
						<span class="block text-[9px] font-bold tracking-wider text-slate-400 uppercase"
							>Certificates</span
						>
						<span class="mt-0.5 block text-xs font-bold text-[#881B1B]"
							>{activeStudent.certificates} uploaded</span
						>
					</div>
					<div>
						<span class="block text-[9px] font-bold tracking-wider text-slate-400 uppercase"
							>Activities</span
						>
						<span class="mt-0.5 block text-xs font-bold text-slate-800"
							>{activeStudent.activityCount} logged</span
						>
					</div>
				</div>

				<!-- Credits Progress -->
				<div class="space-y-2">
					<div class="flex items-center justify-between">
						<span class="text-[10px] font-bold tracking-wider text-slate-500 uppercase"
							>Credit Progress</span
						>
						<span class="text-[10px] font-extrabold text-slate-700">
							{activeStudent.creditsEarned} / {activeStudent.creditsTarget}
						</span>
					</div>
					<div class="h-2.5 w-full overflow-hidden rounded-full bg-slate-100">
						<div
							class="h-full rounded-full transition-all duration-500 {activeStudent.creditsEarned >=
							150
								? 'bg-emerald-500'
								: activeStudent.creditsEarned >= 100
									? 'bg-amber-500'
									: 'bg-rose-500'}"
							style="width: {Math.min(
								100,
								(activeStudent.creditsEarned / activeStudent.creditsTarget) * 100
							)}%"
						></div>
					</div>
					<div class="flex justify-between px-0.5 text-[9px] font-bold text-slate-400">
						<span>0</span>
						<span>50</span>
						<span>100</span>
						<span>150</span>
						<span>200 (Target)</span>
					</div>
				</div>
			</div>

			<!-- Modal Footer -->
			<div class="flex items-center justify-end gap-3 border-t border-slate-100 bg-slate-50/20 p-4">
				<button
					onclick={closeModal}
					class="rounded-lg border border-slate-200 px-4 py-2 text-xs font-bold text-slate-600 transition-colors hover:bg-slate-100 focus:outline-none"
				>
					Close
				</button>
				<button
					onclick={handleSendNotice}
					class="inline-flex items-center gap-1.5 px-4 py-2 text-xs font-bold text-white bg-[#881B1B] hover:bg-[#881B1B]/90 rounded-lg transition-colors focus:outline-none shadow-xs"
				>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						fill="none"
						viewBox="0 0 24 24"
						stroke-width="2"
						stroke="currentColor"
						class="h-3.5 w-3.5"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M14.857 17.082a23.848 23.848 0 0 0 5.454-1.31A8.967 8.967 0 0 1 18 9.75V9A6 6 0 0 0 6 9v.75a8.967 8.967 0 0 1-2.312 6.022c1.733.64 3.56 1.085 5.455 1.31m5.714 0a24.255 24.255 0 0 1-5.714 0m5.714 0a3 3 0 1 1-5.714 0"
						/>
					</svg>
					Send Notice
				</button>
				<button
					onclick={() => activeStudent && openStudentDetail(activeStudent)}
					class="inline-flex items-center gap-1.5 rounded-lg bg-[#881B1B] px-4 py-2 text-xs font-bold text-white shadow-xs transition-colors hover:bg-[#881B1B]/90 focus:outline-none"
				>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						fill="none"
						viewBox="0 0 24 24"
						stroke-width="2"
						stroke="currentColor"
						class="h-3.5 w-3.5"
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
					View Details
				</button>
			</div>
		</div>
	</div>
{/if}
