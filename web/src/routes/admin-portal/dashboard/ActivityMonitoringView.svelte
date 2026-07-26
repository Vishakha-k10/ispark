<script lang="ts">
	import { fade, slide } from 'svelte/transition';
	import { API_BASE_URL } from '$lib/config';

	// Reactive filter states
	let searchQuery = $state('');
	let selectedCategory = $state('');
	let selectedStatus = $state('');
	let sortBy = $state('');

	let loading = $state(true);

	let stats = $state({
		total_monitored: 0,
		total_monitored_change: '',
		ongoing: 0,
		nearing_deadline: '',
		completed: 0,
		completed_change: '',
		pending_verification: 0,
		requires_review: ''
	});

	interface ActivityItem {
		id: number;
		name: string;
		category: string;
		credits: number;
		registered: number;
		completed: number;
		startDate: string;
		endDate: string;
		status: string;
		coordinator?: string;
		description?: string;
	}

	interface StudentItem {
		id: number;
		name: string;
		initials: string;
		enrollment: string;
		activity: string;
		issue: string;
		statusColor: string;
		days: number;
		courseName?: string;
		course_name?: string;
		creditsEarned?: number;
		credits_earned?: number;
	}

	interface MonitoringInsights {
		most_participated: { name: string; count: number } | null;
		highest_credit: { name: string; credits: number } | null;
		highest_completion: { name: string; rate: number } | null;
		students_requiring_followup: { count: number; activity_count: number } | null;
		pending_certificates: number;
		below_credit_target: number;
		inactive_students: number;
		activities_pending_review: number;
	}

	let insights = $state<MonitoringInsights>({
		most_participated: null,
		highest_credit: null,
		highest_completion: null,
		students_requiring_followup: { count: 0, activity_count: 0 },
		pending_certificates: 0,
		below_credit_target: 0,
		inactive_students: 0,
		activities_pending_review: 0
	});

	let activities = $state<ActivityItem[]>([]);

	let attentionStudents = $state<StudentItem[]>([]);

	// Modals & Details states
	let isViewModalOpen = $state(false);
	let isManageModalOpen = $state(false);
	let isStudentModalOpen = $state(false);
	let activeActivity = $state<ActivityItem | null>(null);
	let activeStudent = $state<StudentItem | null>(null);

	// Toasts notifications states
	interface Toast {
		id: number;
		message: string;
		type?: 'success' | 'error';
	}
	let toasts = $state<Toast[]>([]);
	let toastCounter = 0;

	function triggerToast(message: string, type: 'success' | 'error' = 'success') {
		const id = toastCounter++;
		toasts = [...toasts, { id, message, type }];
		setTimeout(() => {
			toasts = toasts.filter((t) => t.id !== id);
		}, 3500);
	}

	async function fetchStats() {
		const token = localStorage.getItem('admin_token');
		try {
			const res = await fetch(`${API_BASE_URL}/api/admin/monitoring/stats`, {
				headers: token ? { Authorization: `Bearer ${token}` } : {}
			});
			if (res.ok) {
				const data = await res.json();
				stats = data;
			}
		} catch {
			// keep fallback state
		}
	}

	async function fetchActivities() {
		const token = localStorage.getItem('admin_token');
		loading = true;
		try {
			/* eslint-disable-next-line svelte/prefer-svelte-reactivity */
			const params = new URLSearchParams();
			if (searchQuery.trim()) params.append('search', searchQuery.trim());
			if (selectedCategory && selectedCategory !== 'All Categories')
				params.append('category', selectedCategory);
			if (selectedStatus && selectedStatus !== 'All Statuses')
				params.append('status', selectedStatus);
			if (sortBy && sortBy !== 'default') params.append('sort_by', sortBy);

			const res = await fetch(`${API_BASE_URL}/api/admin/monitoring/activities?${params}`, {
				headers: token ? { Authorization: `Bearer ${token}` } : {}
			});

			if (res.ok) {
				const data = await res.json();
				activities = data;
			}
		} catch {
			// keep current activities state
		} finally {
			loading = false;
		}
	}

	async function fetchInsights() {
		const token = localStorage.getItem('admin_token');
		try {
			const res = await fetch(`${API_BASE_URL}/api/admin/monitoring/insights`, {
				headers: token ? { Authorization: `Bearer ${token}` } : {}
			});
			if (res.ok) {
				const data = await res.json();
				insights = data;
			}
		} catch {
			// keep fallback state
		}
	}

	async function fetchAttentionStudents() {
		const token = localStorage.getItem('admin_token');
		try {
			const res = await fetch(`${API_BASE_URL}/api/admin/monitoring/attention-students`, {
				headers: token ? { Authorization: `Bearer ${token}` } : {}
			});
			if (res.ok) {
				const data = await res.json();
				attentionStudents = data;
			}
		} catch {
			// keep fallback state
		}
	}

	$effect(() => {
		fetchStats();
		fetchActivities();
		fetchInsights();
		fetchAttentionStudents();
	});

	// Derived metrics
	/* eslint-disable-next-line @typescript-eslint/no-unused-vars */
	let totalRegisteredStudents = $derived(activities.reduce((sum, a) => sum + a.registered, 0));
	/* eslint-disable-next-line @typescript-eslint/no-unused-vars */
	let totalCompletedStudents = $derived(activities.reduce((sum, a) => sum + a.completed, 0));
	/* eslint-disable-next-line @typescript-eslint/no-unused-vars */
	let totalCreditsAwarded = $derived(
		activities.reduce((sum, a) => sum + a.credits * a.completed, 0)
	);

	// Filter and Sort Computing
	let filteredActivities = $derived.by(() => {
		let list = [...activities];

		if (searchQuery.trim() !== '') {
			const query = searchQuery.toLowerCase();
			list = list.filter(
				(act) =>
					act.name.toLowerCase().includes(query) || act.category.toLowerCase().includes(query)
			);
		}

		if (selectedCategory !== '' && selectedCategory !== 'All Categories') {
			const catLower = selectedCategory.toLowerCase();
			list = list.filter(
				(act) =>
					act.category.toLowerCase().includes(catLower) ||
					catLower.includes(act.category.toLowerCase())
			);
		}

		if (selectedStatus !== '' && selectedStatus !== 'All Statuses') {
			const statLower = selectedStatus.toLowerCase();
			list = list.filter(
				(act) =>
					act.status.toLowerCase().includes(statLower) ||
					statLower.includes(act.status.toLowerCase())
			);
		}

		if (sortBy === 'credits-desc') {
			list.sort((a, b) => b.credits - a.credits);
		} else if (sortBy === 'credits-asc') {
			list.sort((a, b) => a.credits - b.credits);
		} else if (sortBy === 'registered-desc') {
			list.sort((a, b) => b.registered - a.registered);
		} else if (sortBy === 'completion-desc') {
			list.sort(
				(a, b) =>
					(b.registered > 0 ? b.completed / b.registered : 0) -
					(a.registered > 0 ? a.completed / a.registered : 0)
			);
		}

		return list;
	});

	/* eslint-disable-next-line @typescript-eslint/no-unused-vars */
	function resetFilters() {
		searchQuery = '';
		selectedCategory = 'All Categories';
		selectedStatus = 'All Statuses';
		sortBy = 'default';
	}

	function handleViewActivity(act: ActivityItem) {
		activeActivity = act;
		isViewModalOpen = true;
	}

	function handleManageActivity(act: ActivityItem) {
		activeActivity = { ...act }; // copy state for editing
		isManageModalOpen = true;
	}

	async function handleSaveActivityChanges() {
		if (!activeActivity) return;
		const token = localStorage.getItem('admin_token');
		try {
			const res = await fetch(
				`${API_BASE_URL}/api/admin/monitoring/activities/${activeActivity.id}`,
				{
					method: 'PUT',
					headers: {
						'Content-Type': 'application/json',
						Authorization: `Bearer ${token}`
					},
					body: JSON.stringify({
						name: activeActivity.name,
						category: activeActivity.category,
						credits: activeActivity.credits,
						status: activeActivity.status,
						startDate: activeActivity.startDate,
						endDate: activeActivity.endDate,
						coordinator: activeActivity.coordinator
					})
				}
			);

			if (res.ok) {
				const data = await res.json();
				triggerToast(
					data.message || `Activity "${activeActivity.name}" updated successfully.`,
					'success'
				);
				await fetchActivities();
				await fetchStats();
				isManageModalOpen = false;
			} else {
				const errData = await res.json().catch(() => ({}));
				triggerToast(errData.error || errData.message || 'Failed to update activity', 'error');
			}
		} catch {
			triggerToast(`Network error: Failed to update activity "${activeActivity.name}".`, 'error');
		}
	}

	async function handleSendReminder(student: StudentItem) {
		const token = localStorage.getItem('admin_token');
		try {
			const res = await fetch(`${API_BASE_URL}/api/admin/monitoring/send-reminder`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					Authorization: `Bearer ${token}`
				},
				body: JSON.stringify({
					student_enrollment: student.enrollment,
					activity_name: student.activity,
					issue: student.issue
				})
			});

			if (res.ok) {
				const data = await res.json();
				triggerToast(data.message || `Reminder alert sent to ${student.name}.`, 'success');
			} else {
				const errData = await res.json().catch(() => ({}));
				triggerToast(
					errData.error || errData.message || `Failed to send reminder alert to ${student.name}`,
					'error'
				);
			}
		} catch {
			triggerToast(`Network error: Failed to send reminder to ${student.name}.`, 'error');
		}
	}

	function handleViewStudent(student: StudentItem) {
		activeStudent = student;
		isStudentModalOpen = true;
	}

	function scrollToAttention() {
		const target = document.getElementById('students-attention');
		if (target) {
			target.scrollIntoView({ behavior: 'smooth' });
		}
	}
</script>

<!-- Toasts container -->
<div class="fixed bottom-6 right-6 z-50 flex flex-col gap-2 max-w-sm">
	{#each toasts as toast (toast.id)}
		<div
			transition:slide={{ duration: 150 }}
			class={`p-4 border rounded-xl shadow-2xl flex items-center gap-2 text-xs font-semibold font-sans ${
				toast.type === 'error'
					? 'bg-slate-900 border-rose-500/60 text-rose-200'
					: 'bg-slate-800 border-slate-700 text-white'
			}`}
		>
			{#if toast.type === 'error'}
				<svg
					xmlns="http://www.w3.org/2000/svg"
					fill="none"
					viewBox="0 0 24 24"
					stroke-width="2"
					stroke="currentColor"
					class="w-4 h-4 text-rose-400 shrink-0"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M12 9v3.75m9-.75a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-9 3.75h.008v.008H12v-.008Z"
					/>
				</svg>
			{:else}
				<svg
					xmlns="http://www.w3.org/2000/svg"
					fill="none"
					viewBox="0 0 24 24"
					stroke-width="2"
					stroke="currentColor"
					class="w-4 h-4 text-accent-gold shrink-0"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M12 9v3.75m0-10.036A11.959 11.959 0 0 1 3.598 6 11.99 11.99 0 0 0 3 9.75c0 5.592 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.57-.598-3.75h-.152c-3.196 0-6.1-1.249-8.25-3.286Zm0 13.036h.008v.008H12v-.008Z"
					/>
				</svg>
			{/if}
			<span>{toast.message}</span>
		</div>
	{/each}
</div>

<!-- ==================== OVERVIEW STATS (KPI GRID) ==================== -->
<section
	class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-5"
	aria-label="Activity statistics overview"
>
	<!-- Stat Card 1: Total Activities -->
	<div
		class="bg-white p-5 rounded-xl border border-slate-200 flex flex-col justify-between shadow-xs hover:shadow-md transition-shadow"
	>
		<div class="flex items-center justify-between">
			<span class="text-2xl font-bold font-serif text-slate-900">{stats.total_monitored}</span>
			<div class="p-2.5 rounded-lg bg-blue-50 text-blue-600 border border-blue-100">
				<!-- Chart line icon -->
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
			<h3 class="text-xs font-bold text-slate-900 capitalize">
				Total Activities Monitored
			</h3>
			<span class="text-[10px] font-bold text-[#8fa3bb] uppercase tracking-wider mt-0.5 block">{stats.total_monitored_change}</span>
		</div>
	</div>

	<!-- Stat Card 2: Ongoing Activities -->
	<div
		class="bg-white p-5 rounded-xl border border-slate-200 flex flex-col justify-between shadow-xs hover:shadow-md transition-shadow"
	>
		<div class="flex items-center justify-between">
			<span class="text-2xl font-bold font-serif text-slate-900">{stats.ongoing}</span>
			<div class="p-2.5 rounded-lg bg-amber-50 text-amber-600 border border-amber-100">
				<!-- Clock icon -->
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
			<h3 class="text-xs font-bold text-slate-900 capitalize">
				Ongoing Activities
			</h3>
			<span class="text-[10px] font-bold text-[#8fa3bb] uppercase tracking-wider mt-0.5 block">
				{stats.nearing_deadline}
			</span>
		</div>
	</div>

	<!-- Stat Card 3: Completed Activities -->
	<div
		class="bg-white p-5 rounded-xl border border-slate-200 flex flex-col justify-between shadow-xs hover:shadow-md transition-shadow"
	>
		<div class="flex items-center justify-between">
			<span class="text-2xl font-bold font-serif text-slate-900">{stats.completed}</span>
			<div class="p-2.5 rounded-lg bg-emerald-50 text-emerald-600 border border-emerald-100">
				<!-- Checkmark Shield icon -->
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
						d="M9 12.75 11.25 15 15 9.75M21 12c0 1.268-.63 2.39-1.593 3.068a3.745 3.745 0 0 1-1.043 3.296 3.745 3.745 0 0 1-3.296 1.043A3.745 3.745 0 0 1 12 21c-1.268 0-2.39-.63-3.068-1.593a3.746 3.746 0 0 1-3.296-1.043 3.745 3.745 0 0 1-1.043-3.296A3.745 3.745 0 0 1 3 12c0-1.268.63-2.39 1.593-3.068a3.745 3.745 0 0 1 1.043-3.296 3.746 3.746 0 0 1 3.296-1.043A3.746 3.746 0 0 1 12 3c1.268 0 2.39.63 3.068 1.593a3.746 3.746 0 0 1 3.296 1.043 3.746 3.746 0 0 1 1.043 3.296A3.745 3.745 0 0 1 21 12Z"
					/>
				</svg>
			</div>
		</div>
		<div class="mt-4">
			<h3 class="text-xs font-bold text-slate-900 capitalize">
				Completed Activities
			</h3>
			<span class="text-[10px] font-bold text-[#8fa3bb] uppercase tracking-wider mt-0.5 block">
    			{stats.completed_change}
			</span>
		</div>
	</div>

	<!-- Stat Card 4: Pending Verification -->
	<div
		class="bg-white p-5 rounded-xl border border-slate-200 flex flex-col justify-between shadow-xs hover:shadow-md transition-shadow"
	>
		<div class="flex items-center justify-between">
			<span class="text-2xl font-bold font-serif text-slate-900">{stats.pending_verification}</span>
			<div class="p-2.5 rounded-lg bg-rose-50 text-rose-600 border border-rose-100">
				<!-- Exclamation warning icon -->
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
						d="M12 9v3.75m9-.75a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-9 3.75h.008v.008H12v-.008Z"
					/>
				</svg>
			</div>
		</div>
		<div class="mt-4">
			<h3 class="text-xs font-bold text-slate-900 capitalize">
				Pending Verification
			</h3>
			<span class="text-[10px] font-bold text-[#8fa3bb] uppercase tracking-wider mt-0.5 block">
				{stats.requires_review}
			</span>
		</div>
	</div>
</section>

<!-- ==================== ACTIVITY REGISTRY TABLE ==================== -->
<section class="bg-white border border-slate-200 rounded-xl shadow-xs overflow-hidden">
	<div class="p-5 border-b border-slate-100 flex items-center justify-between bg-slate-50/20">
		<h2 class="text-sm font-bold font-serif text-slate-900">Activity Monitoring</h2>
		<span
			class="bg-slate-100 border border-slate-200 text-slate-600 text-[10px] font-extrabold uppercase px-2.5 py-1 rounded"
		>
			{filteredActivities.length} Activities
		</span>
	</div>

	<!-- Filter Inputs bar -->
	<div class="p-4 border-b border-slate-100 flex flex-col sm:flex-row gap-3 bg-slate-50/10">
		<div class="flex-grow relative">
			<input
				type="text"
				bind:value={searchQuery}
				oninput={() => fetchActivities()}
				placeholder="Search Activity..."
				class="w-full pl-9 pr-4 py-2 border border-slate-200 rounded-lg text-xs text-slate-800 bg-white focus:outline-none focus:border-slate-355"
			/>
			<span class="absolute left-3 top-2.5 text-slate-400">
				<svg
					xmlns="http://www.w3.org/2000/svg"
					fill="none"
					viewBox="0 0 24 24"
					stroke-width="2"
					stroke="currentColor"
					class="w-4 h-4"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="m21 21-5.197-5.197m0 0A7.5 7.5 0 1 0 5.196 5.196a7.5 7.5 0 0 0 10.637 10.637Z"
					/>
				</svg>
			</span>
		</div>

		<!-- Category dropdown -->
		<select
			bind:value={selectedCategory}
			onchange={() => fetchActivities()}
			class="px-3 py-2 border border-slate-200 rounded-lg text-xs text-slate-700 bg-white focus:outline-none focus:border-slate-355"
		>
			<option value="">All Categories</option>
			<option value="Leadership">Leadership</option>
			<option value="Cultural">Cultural</option>
			<option value="Public Speaking">Public Speaking</option>
			<option value="Social Service">Social Service</option>
			<option value="Technical">Technical</option>
			<option value="Sports">Sports</option>
			<option value="Soft Skills">Soft Skills</option>
			<option value="Academic">Academic</option>
			<option value="Co-curricular">Co-curricular</option>
			<option value="Industry">Industry</option>
		</select>

		<!-- Status dropdown -->
		<select
			bind:value={selectedStatus}
			onchange={() => fetchActivities()}
			class="px-3 py-2 border border-slate-200 rounded-lg text-xs text-slate-700 bg-white focus:outline-none focus:border-slate-355"
		>
			<option value="">All Statuses</option>
			<option value="Completed">Completed</option>
			<option value="Ongoing">Ongoing</option>
			<option value="Pending Verification">Pending Verification</option>
			<option value="Upcoming">Upcoming</option>
		</select>

		<!-- Sorting dropdown -->
		<select
			bind:value={sortBy}
			onchange={() => fetchActivities()}
			class="px-3 py-2 border border-slate-200 rounded-lg text-xs text-slate-700 bg-white focus:outline-none focus:border-slate-355"
		>
			<option value="">Sort By</option>
			<option value="credits-desc">Credits: High to Low</option>
			<option value="credits-asc">Credits: Low to High</option>
			<option value="registered-desc">Registered: High to Low</option>
			<option value="completion-desc">Completion %: High to Low</option>
		</select>
	</div>

	<!-- Registry Table -->
	<div class="overflow-x-auto">
		<table class="w-full text-left border-collapse">
			<thead>
				<tr
					class="border-b border-slate-150 bg-slate-50/50 text-[10px] font-extrabold text-slate-405 uppercase tracking-wider"
				>
					<th class="py-3.5 px-5">Activity Name</th>
					<th class="py-3.5 px-5">Category</th>
					<th class="py-3.5 px-5">Credits</th>
					<th class="py-3.5 px-5">Registered Students</th>
					<th class="py-3.5 px-5">Completed Students</th>
					<th class="py-3.5 px-5">Completion %</th>
					<th class="py-3.5 px-5">Start Date</th>
					<th class="py-3.5 px-5">End Date</th>
					<th class="py-3.5 px-5">Status</th>
					<th class="py-3.5 px-5 text-center">Actions</th>
				</tr>
			</thead>
			<tbody class="divide-y divide-slate-100 text-xs font-sans">
				{#if loading}
					<tr>
						<td colspan="10" class="py-8 text-center text-slate-400 font-semibold">
							<span class="inline-flex items-center gap-2">
								<svg
									class="animate-spin h-4 w-4 text-slate-400"
									xmlns="http://www.w3.org/2000/svg"
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
								Loading activities...
							</span>
						</td>
					</tr>
				{:else if filteredActivities.length > 0}
					{#each filteredActivities as act (act.id)}
						{@const progress =
							act.registered > 0 ? Math.round((act.completed / act.registered) * 100) : 0}
						<tr class="hover:bg-slate-50/40 transition-colors">
							<td class="py-4 px-5 font-bold text-slate-850">{act.name}</td>
							<td class="py-4 px-5 text-slate-500 font-semibold">{act.category}</td>
							<td class="py-4 px-5 font-extrabold text-slate-900">{act.credits}</td>
							<td class="py-4 px-5 text-slate-600 font-semibold">{act.registered}</td>
							<td class="py-4 px-5 text-slate-600 font-semibold">{act.completed}</td>
							<td class="py-4 px-5">
								<div class="flex items-center gap-2">
									<div class="w-16 h-1.5 bg-slate-100 rounded-full overflow-hidden shrink-0">
										<div
											class="h-full rounded-full transition-all duration-300
											{progress >= 85 ? 'bg-emerald-600' : progress >= 40 ? 'bg-amber-500' : 'bg-slate-350'}"
											style="width: {progress}%"
										></div>
									</div>
									<span class="font-extrabold text-slate-700 text-[10px] w-6 shrink-0"
										>{progress}%</span
									>
								</div>
							</td>
							<td class="py-4 px-5 text-slate-500 font-semibold">{act.startDate}</td>
							<td class="py-4 px-5 text-slate-500 font-semibold">{act.endDate}</td>
							<td class="py-4 px-5">
								<span
									class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md text-[10px] font-extrabold uppercase tracking-wide border
									{act.status === 'Completed'
										? 'bg-emerald-50 text-emerald-700 border-emerald-100'
										: act.status === 'Ongoing'
											? 'bg-blue-50 text-blue-700 border-blue-100'
											: act.status === 'Pending Verification'
												? 'bg-amber-50 text-amber-705 border-amber-100'
												: 'bg-slate-50 text-slate-600 border-slate-200'}"
								>
									<span
										class="w-1.5 h-1.5 rounded-full
										{act.status === 'Completed'
											? 'bg-emerald-600'
											: act.status === 'Ongoing'
												? 'bg-blue-600'
												: act.status === 'Pending Verification'
													? 'bg-amber-500'
													: 'bg-slate-400'}"
									></span>
									{act.status}
								</span>
							</td>
							<td class="py-4 px-5">
								<div class="flex items-center justify-center gap-2">
									<button
										onclick={() => handleViewActivity(act)}
										class="inline-flex items-center gap-1.5 px-3 py-1.5 bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 font-bold text-[10px] uppercase rounded-lg transition-colors focus:outline-none"
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
										View
									</button>
									<button
										onclick={() => handleManageActivity(act)}
										class="inline-flex items-center gap-1.5 px-3 py-1.5 bg-accent-red text-white hover:bg-accent-red/90 font-extrabold text-[10px] uppercase rounded-lg transition-colors focus:outline-none"
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
												d="M9.594 3.94c.09-.542.56-.94 1.11-.94h2.593c.55 0 1.02.398 1.11.94l.213 1.281c.063.374.313.686.645.87.074.04.147.083.22.127.324.196.72.257 1.075.124l1.217-.456a1.125 1.125 0 0 1 1.37.49l1.296 2.247a1.125 1.125 0 0 1-.26 1.43l-1.003.828c-.293.241-.438.613-.43.992a7.723 7.723 0 0 1 0 .255c-.008.378.137.75.43.991l1.004.827c.424.35.534.954.26 1.43l-1.298 2.247a1.125 1.125 0 0 1-1.369.491l-1.217-.456c-.355-.133-.75-.072-1.076.124a6.47 6.47 0 0 1-.22.128c-.331.183-.581.495-.644.869l-.213 1.281c-.09.543-.56.94-1.11.94h-2.594c-.55 0-1.019-.398-1.11-.94l-.213-1.281c-.062-.374-.312-.686-.644-.87a6.52 6.52 0 0 1-.22-.127c-.325-.196-.72-.257-1.076-.124l-1.217.456a1.125 1.125 0 0 1-1.369-.49l-1.297-2.247a1.125 1.125 0 0 1 .26-1.43l1.004-.827c.292-.24.437-.613.43-.991a6.936 6.936 0 0 1 0-.255c.007-.38-.138-.751-.43-.992l-1.004-.827a1.125 1.125 0 0 1-.26-1.43l1.297-2.247a1.125 1.125 0 0 1 1.37-.491l1.216.456c.356.133.751.072 1.076-.124.072-.044.146-.086.22-.128.332-.183.582-.495.644-.869l.214-1.28Z"
											/>
											<path
												stroke-linecap="round"
												stroke-linejoin="round"
												d="M15 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z"
											/>
										</svg>
										Manage
									</button>
								</div>
							</td>
						</tr>
					{/each}
				{:else}
					<tr>
						<td colspan="10" class="py-8 text-center text-slate-400 font-semibold">
							No activities found matching your criteria.
						</td>
					</tr>
				{/if}
			</tbody>
		</table>
	</div>

	<div
		class="p-4 border-t border-slate-100 bg-slate-50/30 text-slate-500 font-semibold text-[11px] flex justify-between"
	>
		<span>Showing {filteredActivities.length} of {activities.length} activities</span>
	</div>
</section>

<!-- ==================== PERFORMANCE SNAPSHOT & QUICK INSIGHTS ==================== -->
<div class="grid grid-cols-1 md:grid-cols-2 gap-6 items-start">
	<!-- Activity Performance Snapshot Card -->
	<section class="bg-white border border-slate-200 rounded-xl p-5 shadow-xs space-y-4">
		<h2 class="text-sm font-bold font-serif text-slate-900">Activity Performance Snapshot</h2>
		<div class="h-px bg-slate-100 my-2"></div>

		<div class="space-y-3.5">
			<!-- Snapshot Item 1 -->
			<div
				class="flex items-center gap-3 p-3 border border-slate-100 rounded-lg hover:bg-slate-50/50 transition-colors"
			>
				<div class="p-2.5 bg-blue-50 text-blue-600 border border-blue-100 rounded-lg shrink-0">
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
							d="M3.75 3v11.25A2.25 2.25 0 0 0 6 16.5h2.25M3.75 3h-1.5m1.5 0h16.5m0 0h1.5m-1.5 0v11.25A2.25 2.25 0 0 1 18 16.5h-2.25m-7.5 0h7.5m-7.5 0-1 3m8.5-3 1 3m0 0 .5 1.5m-.5-1.5h-9.5m0 0-.5 1.5m.75-9 3-3 2.148 2.148A12.061 12.061 0 0 1 16.5 7.605"
						/>
					</svg>
				</div>
				<div>
					<span class="text-[9px] font-bold text-slate-400 uppercase tracking-wider block"
						>Most Participated Activity</span
					>
					<span class="text-xs font-bold text-slate-800 block mt-0.5"
						>{insights?.most_participated?.name || 'N/A'}</span
					>
					<span class="text-[10px] text-slate-500 font-semibold block mt-0.5"
						>{insights?.most_participated?.count ?? 0} students</span
					>
				</div>
			</div>

			<!-- Snapshot Item 2 -->
			<div
				class="flex items-center gap-3 p-3 border border-slate-100 rounded-lg hover:bg-slate-50/50 transition-colors"
			>
				<div class="p-2.5 bg-amber-50 text-amber-600 border border-amber-100 rounded-lg shrink-0">
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
							d="M16.5 6v.75m0 3v.75m0 3v.75m0 3V18m-3-12v.75m0 3v.75m0 3v.75m0 3V18m-6-12v.75m0 3v.75m0 3v.75m0 3V18m-3-12v.75m0 3v.75m0 3v.75m0 3V18M3 6.75A.75.75 0 0 1 3.75 6h16.5a.75.75 0 0 1 .75.75v10.5a.75.75 0 0 1-.75.75H3.75a.75.75 0 0 1-.75-.75V6.75Z"
						/>
					</svg>
				</div>
				<div>
					<span class="text-[9px] font-bold text-slate-400 uppercase tracking-wider block"
						>Highest Credit Activity</span
					>
					<span class="text-xs font-bold text-slate-800 block mt-0.5"
						>{insights?.highest_credit?.name || 'N/A'}</span
					>
					<span class="text-[10px] text-slate-500 font-semibold block mt-0.5"
						>{insights?.highest_credit?.credits ?? 0} credits</span
					>
				</div>
			</div>

			<!-- Snapshot Item 3 -->
			<div
				class="flex items-center gap-3 p-3 border border-slate-100 rounded-lg hover:bg-slate-50/50 transition-colors"
			>
				<div
					class="p-2.5 bg-emerald-50 text-emerald-600 border border-emerald-100 rounded-lg shrink-0"
				>
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
							d="M9 12.75 11.25 15 15 9.75M21 12c0 1.268-.63 2.39-1.593 3.068a3.745 3.745 0 0 1-1.043 3.296 3.745 3.745 0 0 1-3.296 1.043A3.745 3.745 0 0 1 12 21c-1.268 0-2.39-.63-3.068-1.593a3.746 3.746 0 0 1-3.296-1.043 3.745 3.745 0 0 1-1.043-3.296A3.745 3.745 0 0 1 3 12c0-1.268.63-2.39 1.593-3.068a3.745 3.745 0 0 1 1.043-3.296 3.746 3.746 0 0 1 3.296-1.043A3.746 3.746 0 0 1 12 3c1.268 0 2.39.63 3.068 1.593a3.746 3.746 0 0 1 3.296 1.043 3.746 3.746 0 0 1 1.043 3.296A3.745 3.745 0 0 1 21 12Z"
						/>
					</svg>
				</div>
				<div>
					<span class="text-[9px] font-bold text-slate-400 uppercase tracking-wider block"
						>Highest Completion Rate</span
					>
					<span class="text-xs font-bold text-slate-800 block mt-0.5"
						>{insights?.highest_completion?.name || 'N/A'}</span
					>
					<span class="text-[10px] text-slate-500 font-semibold block mt-0.5"
						>{insights?.highest_completion?.rate ?? 0}% completion</span
					>
				</div>
			</div>

			<!-- Snapshot Item 4 -->
			<div
				class="flex items-center gap-3 p-3 border border-slate-100 rounded-lg hover:bg-slate-50/50 transition-colors"
			>
				<div class="p-2.5 bg-rose-50 text-rose-600 border border-rose-100 rounded-lg shrink-0">
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
							d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126ZM12 15.75h.007v.008H12v-.008Z"
						/>
					</svg>
				</div>
				<div>
					<span class="text-[9px] font-bold text-slate-400 uppercase tracking-wider block"
						>Students Requiring Follow-up</span
					>
					<span class="text-xs font-bold text-slate-800 block mt-0.5"
						>{insights?.students_requiring_followup?.count ?? 0} students</span
					>
					<span class="text-[10px] text-slate-500 font-semibold block mt-0.5"
						>across {insights?.students_requiring_followup?.activity_count ?? 0} activities</span
					>
				</div>
			</div>
		</div>
	</section>

	<!-- Quick Insights Card -->
	<section
		class="bg-white border border-slate-200 rounded-xl p-5 shadow-xs space-y-4 flex flex-col justify-between h-full"
	>
		<div class="space-y-4">
			<div>
				<h2 class="text-sm font-bold font-serif text-slate-900">Quick Insights</h2>
				<p class="text-[10px] font-bold text-slate-400 uppercase tracking-wider mt-0.5">
					Students needing attention
				</p>
			</div>
			<div class="h-px bg-slate-100 my-2"></div>

			<div class="divide-y divide-slate-100 text-xs font-semibold text-slate-700 font-sans">
				<div class="py-3 flex items-center justify-between">
					<span>Students with pending certificates</span>
					<span
						class="w-7 h-7 bg-rose-50 border border-rose-100 text-rose-700 font-extrabold flex items-center justify-center rounded-full"
						>{insights.pending_certificates}</span
					>
				</div>
				<div class="py-3 flex items-center justify-between">
					<span>Students below credit target</span>
					<span
						class="w-7 h-7 bg-amber-50 border border-amber-100 text-amber-700 font-extrabold flex items-center justify-center rounded-full"
						>{insights.below_credit_target}</span
					>
				</div>
				<div class="py-3 flex items-center justify-between">
					<span>Inactive students (30+ days)</span>
					<span
						class="w-7 h-7 bg-slate-100 border border-slate-250 text-slate-700 font-extrabold flex items-center justify-center rounded-full"
						>{insights.inactive_students}</span
					>
				</div>
				<div class="py-3 flex items-center justify-between">
					<span>Activities pending review</span>
					<span
						class="w-7 h-7 bg-slate-100 border border-slate-250 text-slate-700 font-extrabold flex items-center justify-center rounded-full"
						>{insights.activities_pending_review}</span
					>
				</div>
			</div>
		</div>

		<div class="pt-4 border-t border-slate-100">
			<button
				onclick={scrollToAttention}
				class="w-full py-2.5 text-center bg-[#881B1B] hover:bg-[#881B1B]/90 text-white font-bold text-[11px] uppercase tracking-wider rounded-lg transition-colors focus:outline-none"
			>
				View Students Requiring Attention
			</button>
		</div>
	</section>
</div>

<!-- ==================== STUDENTS REQUIRING ATTENTION TABLE ==================== -->
<section
	id="students-attention"
	class="bg-white border border-slate-200 rounded-xl shadow-xs overflow-hidden scroll-mt-20"
>
	<div class="p-5 border-b border-slate-100 bg-slate-50/20">
		<h2 class="text-sm font-bold font-serif text-slate-900">Students Requiring Attention</h2>
	</div>

	<div class="overflow-x-auto">
		<table class="w-full text-left border-collapse">
			<thead>
				<tr
					class="border-b border-slate-150 bg-slate-50/50 text-[10px] font-extrabold text-slate-405 uppercase tracking-wider"
				>
					<th class="py-3.5 px-5">Student Name</th>
					<th class="py-3.5 px-5">Enrollment Number</th>
					<th class="py-3.5 px-5">Activity</th>
					<th class="py-3.5 px-5">Issue</th>
					<th class="py-3.5 px-5">Days Pending</th>
					<th class="py-3.5 px-5 text-center">Action</th>
				</tr>
			</thead>
			<tbody class="divide-y divide-slate-100 text-xs font-sans">
				{#if loading}
					<tr>
						<td colspan="6" class="py-8 text-center text-slate-400 font-semibold">
							<span class="inline-flex items-center gap-2">
								<svg
									class="animate-spin h-4 w-4 text-slate-400"
									xmlns="http://www.w3.org/2000/svg"
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
								Loading students...
							</span>
						</td>
					</tr>
				{:else if attentionStudents.length > 0}
					{#each attentionStudents as stud (stud.id)}
						<tr class="hover:bg-slate-50/40 transition-colors">
							<td class="py-4 px-5">
								<div class="flex items-center gap-3">
									<span
										class="w-8 h-8 rounded-full bg-[#881B1B]/10 text-[#881B1B] font-serif font-extrabold flex items-center justify-center shrink-0 border border-[#881B1B]/20 text-[11px]"
									>
										{stud.initials}
									</span>
									<span class="font-bold text-slate-850">{stud.name}</span>
								</div>
							</td>
							<td class="py-4 px-5 font-semibold text-slate-500 select-all">{stud.enrollment}</td>
							<td class="py-4 px-5 font-bold text-slate-700">{stud.activity}</td>
							<td class="py-4 px-5">
								<span class="inline-flex items-center gap-1.5 {stud.statusColor}">
									{#if stud.issue.includes('Not Uploaded') || stud.issue.includes('No Activity')}
										<span class="w-1.5 h-1.5 bg-rose-600 rounded-full shrink-0 animate-pulse"
										></span>
									{:else if stud.issue.includes('Incomplete')}
										<span class="w-1.5 h-1.5 bg-amber-500 rounded-full shrink-0"></span>
									{:else}
										<span class="w-1.5 h-1.5 bg-blue-600 rounded-full shrink-0"></span>
									{/if}
									{stud.issue}
								</span>
							</td>
							<td class="py-4 px-5 font-extrabold text-slate-900">{stud.days} days</td>
							<td class="py-4 px-5">
								<div class="flex items-center justify-center gap-2">
									<button
										onclick={() => handleViewStudent(stud)}
										class="inline-flex items-center gap-1.5 px-3 py-1.5 bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 font-bold text-[10px] uppercase rounded-lg transition-colors focus:outline-none"
									>
										View Profile
									</button>
									<button
										onclick={() => handleSendReminder(stud)}
										class="inline-flex items-center gap-1.5 px-3 py-1.5 bg-white border border-rose-200 hover:bg-rose-50 text-rose-600 font-extrabold text-[10px] uppercase rounded-lg transition-colors focus:outline-none"
									>
										Send Reminder
									</button>
								</div>
							</td>
						</tr>
					{/each}
				{:else}
					<tr>
						<td colspan="6" class="py-8 text-center text-slate-400 font-semibold">
							No students requiring attention at this time.
						</td>
					</tr>
				{/if}
			</tbody>
		</table>
	</div>

	<div
		class="p-4 border-t border-slate-100 bg-slate-50/30 text-slate-500 font-semibold text-[11px]"
	>
		<span>Showing {attentionStudents.length} students requiring attention</span>
	</div>
</section>

<!-- ==================== ACTIVITY DETAILS VIEW MODAL ==================== -->
{#if isViewModalOpen && activeActivity}
	<div
		transition:fade={{ duration: 150 }}
		class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/60 backdrop-blur-xs"
	>
		<!-- svelte-ignore a11y_click_events_have_key_events -->
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<div
			onclick={(e) => e.stopPropagation()}
			class="w-full max-w-lg bg-white border border-slate-200 rounded-2xl shadow-2xl overflow-hidden flex flex-col font-sans"
		>
			<div class="p-5 border-b border-slate-150 flex items-center justify-between bg-slate-50/30">
				<div>
					<h3 class="text-sm font-bold font-serif text-slate-900">Activity Overview Details</h3>
					<p class="text-[9px] font-bold text-slate-400 uppercase tracking-widest mt-0.5">
						ID: ACT-{activeActivity.id}
					</p>
				</div>
				<button
					onclick={() => (isViewModalOpen = false)}
					aria-label="Close modal"
					class="p-1 rounded-lg text-slate-400 hover:bg-slate-100 hover:text-slate-600 transition-colors"
				>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						fill="none"
						viewBox="0 0 24 24"
						stroke-width="2"
						stroke="currentColor"
						class="w-5 h-5"
					>
						<path stroke-linecap="round" stroke-linejoin="round" d="M6 18 18 6M6 6l12 12" />
					</svg>
				</button>
			</div>

			<div class="p-6 space-y-4">
				<div class="grid grid-cols-2 gap-4 bg-slate-50 p-4 rounded-xl border border-slate-150">
					<div>
						<span class="text-[9px] font-bold text-slate-400 uppercase tracking-wider block"
							>Activity Name</span
						>
						<span class="text-xs font-bold text-slate-800 block mt-0.5">{activeActivity.name}</span>
					</div>
					<div>
						<span class="text-[9px] font-bold text-slate-400 uppercase tracking-wider block"
							>Category</span
						>
						<span class="text-xs font-bold text-slate-800 block mt-0.5"
							>{activeActivity.category}</span
						>
					</div>
					<div>
						<span class="text-[9px] font-bold text-slate-400 uppercase tracking-wider block"
							>Credits Offered</span
						>
						<span class="text-xs font-bold text-slate-800 block mt-0.5"
							>{activeActivity.credits} Extracurricular Credits</span
						>
					</div>
					<div>
						<span class="text-[9px] font-bold text-slate-400 uppercase tracking-wider block"
							>Dates</span
						>
						<span class="text-[11px] font-semibold text-slate-600 block mt-0.5"
							>{activeActivity.startDate} - {activeActivity.endDate}</span
						>
					</div>
				</div>

				<div
					class="grid grid-cols-3 gap-3 text-center border border-slate-200 rounded-xl p-3 bg-white"
				>
					<div class="border-r border-slate-100">
						<span class="text-[18px] font-bold text-slate-900 block"
							>{activeActivity.registered}</span
						>
						<span class="text-[9px] font-bold text-slate-400 uppercase tracking-wider mt-0.5 block"
							>Registered</span
						>
					</div>
					<div class="border-r border-slate-100">
						<span class="text-[18px] font-bold text-slate-900 block"
							>{activeActivity.completed}</span
						>
						<span class="text-[9px] font-bold text-slate-400 uppercase tracking-wider mt-0.5 block"
							>Completed</span
						>
					</div>
					<div>
						<span class="text-[18px] font-bold text-emerald-600 block">
							{activeActivity.registered > 0
								? Math.round((activeActivity.completed / activeActivity.registered) * 100)
								: 0}%
						</span>
						<span class="text-[9px] font-bold text-slate-400 uppercase tracking-wider mt-0.5 block"
							>Ratio</span
						>
					</div>
				</div>

				<div class="space-y-1 bg-slate-50/50 p-3.5 border border-slate-150 rounded-xl">
					<span class="text-[9px] font-bold text-slate-400 uppercase tracking-wider block"
						>Co-ordinator / Faculty</span
					>
					<span class="text-xs font-bold text-slate-800 block mt-0.5"
						>{activeActivity.coordinator || 'Faculty Coordinator'}</span
					>
					{#if activeActivity.description}
						<p class="text-[10px] text-slate-500 leading-relaxed mt-1 font-medium">
							{activeActivity.description}
						</p>
					{/if}
				</div>
			</div>

			<div class="p-5 border-t border-slate-150 flex items-center justify-end bg-slate-50/30">
				<button
					onclick={() => (isViewModalOpen = false)}
					class="px-5 py-2 bg-slate-100 hover:bg-slate-200 text-slate-700 font-bold text-xs tracking-wider uppercase rounded-lg transition-colors focus:outline-none"
				>
					Close Overview
				</button>
			</div>
		</div>
	</div>
{/if}

<!-- ==================== ACTIVITY SETTINGS MANAGEMENT MODAL ==================== -->
{#if isManageModalOpen && activeActivity}
	<div
		transition:fade={{ duration: 150 }}
		class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/60 backdrop-blur-xs"
	>
		<!-- svelte-ignore a11y_click_events_have_key_events -->
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<div
			onclick={(e) => e.stopPropagation()}
			class="w-full max-w-lg bg-white border border-slate-200 rounded-2xl shadow-2xl overflow-hidden flex flex-col font-sans"
		>
			<div class="p-5 border-b border-slate-150 flex items-center justify-between bg-slate-50/30">
				<div>
					<h3 class="text-sm font-bold font-serif text-slate-900">Manage Activity Details</h3>
					<p class="text-[9px] font-bold text-slate-400 uppercase tracking-widest mt-0.5">
						Modify parameters for active monitoring
					</p>
				</div>
				<button
					onclick={() => (isManageModalOpen = false)}
					aria-label="Close modal"
					class="p-1 rounded-lg text-slate-400 hover:bg-slate-100 hover:text-slate-600 transition-colors"
				>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						fill="none"
						viewBox="0 0 24 24"
						stroke-width="2"
						stroke="currentColor"
						class="w-5 h-5"
					>
						<path stroke-linecap="round" stroke-linejoin="round" d="M6 18 18 6M6 6l12 12" />
					</svg>
				</button>
			</div>

			<div class="p-6 space-y-4">
				<div class="flex flex-col gap-1.5">
					<label
						for="edit-activity-name"
						class="text-[10px] font-extrabold text-slate-600 tracking-wider">ACTIVITY NAME</label
					>
					<input
						id="edit-activity-name"
						type="text"
						bind:value={activeActivity.name}
						class="px-3 py-2 border border-slate-200 rounded-lg text-xs text-slate-800 focus:outline-none focus:border-slate-355"
					/>
				</div>

				<div class="grid grid-cols-2 gap-4">
					<div class="flex flex-col gap-1.5">
						<label
							for="edit-category"
							class="text-[10px] font-extrabold text-slate-600 tracking-wider">CATEGORY</label
						>
						<select
							id="edit-category"
							bind:value={activeActivity.category}
							class="px-3 py-2 border border-slate-200 rounded-lg text-xs text-slate-800 focus:outline-none focus:border-slate-355"
						>
							<option value="Soft Skills">Soft Skills</option>
							<option value="Academic">Academic</option>
							<option value="Social">Social</option>
							<option value="Technical">Technical</option>
							<option value="Co-curricular">Co-curricular</option>
							<option value="Industry">Industry</option>
							<option value="Sports">Sports</option>
						</select>
					</div>

					<div class="flex flex-col gap-1.5">
						<label
							for="edit-credits"
							class="text-[10px] font-extrabold text-slate-600 tracking-wider">CREDITS</label
						>
						<input
							id="edit-credits"
							type="number"
							bind:value={activeActivity.credits}
							class="px-3 py-2 border border-slate-200 rounded-lg text-xs text-slate-800 focus:outline-none focus:border-slate-355"
						/>
					</div>
				</div>

				<div class="grid grid-cols-2 gap-4">
					<div class="flex flex-col gap-1.5">
						<label
							for="edit-registered"
							class="text-[10px] font-extrabold text-slate-500 tracking-wider flex items-center gap-1"
							>REGISTERED STUDENTS (DERIVED)</label
						>
						<input
							id="edit-registered"
							type="number"
							readonly
							disabled
							value={activeActivity.registered}
							class="px-3 py-2 border border-slate-200 rounded-lg text-xs text-slate-500 bg-slate-100/80 cursor-not-allowed select-none focus:outline-none"
						/>
					</div>

					<div class="flex flex-col gap-1.5">
						<label
							for="edit-completed"
							class="text-[10px] font-extrabold text-slate-500 tracking-wider flex items-center gap-1"
							>COMPLETED STUDENTS (DERIVED)</label
						>
						<input
							id="edit-completed"
							type="number"
							readonly
							disabled
							value={activeActivity.completed}
							class="px-3 py-2 border border-slate-200 rounded-lg text-xs text-slate-500 bg-slate-100/80 cursor-not-allowed select-none focus:outline-none"
						/>
					</div>
				</div>

				<div class="flex flex-col gap-1.5">
					<label for="edit-status" class="text-[10px] font-extrabold text-slate-600 tracking-wider"
						>ACTIVITY STATUS</label
					>
					<select
						id="edit-status"
						bind:value={activeActivity.status}
						class="px-3 py-2 border border-slate-200 rounded-lg text-xs text-slate-800 focus:outline-none focus:border-slate-355"
					>
						<option value="Completed">Completed</option>
						<option value="Ongoing">Ongoing</option>
						<option value="Pending Verification">Pending Verification</option>
						<option value="Upcoming">Upcoming</option>
					</select>
				</div>
			</div>

			<div
				class="p-5 border-t border-slate-150 flex items-center justify-end gap-2.5 bg-slate-50/30"
			>
				<button
					onclick={() => (isManageModalOpen = false)}
					class="px-4 py-2 border border-slate-200 hover:bg-slate-50 text-slate-700 font-bold text-xs uppercase rounded-lg transition-colors focus:outline-none"
				>
					Cancel
				</button>
				<button
					onclick={handleSaveActivityChanges}
					class="px-4 py-2 bg-[#881B1B] hover:bg-[#881B1B]/90 text-white font-bold text-xs uppercase rounded-lg transition-colors focus:outline-none"
				>
					Save Changes
				</button>
			</div>
		</div>
	</div>
{/if}

<!-- ==================== STUDENT DETAILS VIEW MODAL ==================== -->
{#if isStudentModalOpen && activeStudent}
	<div
		transition:fade={{ duration: 150 }}
		class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/60 backdrop-blur-xs"
	>
		<!-- svelte-ignore a11y_click_events_have_key_events -->
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<div
			onclick={(e) => e.stopPropagation()}
			class="w-full max-w-md bg-white border border-slate-200 rounded-2xl shadow-2xl overflow-hidden flex flex-col font-sans"
		>
			<div class="p-5 border-b border-slate-150 flex items-center justify-between bg-slate-50/30">
				<div class="flex items-center gap-3">
					<span
						class="w-9 h-9 rounded-full bg-[#881B1B]/15 text-[#881B1B] font-serif font-extrabold flex items-center justify-center shrink-0 border border-[#881B1B]/30 text-xs"
					>
						{activeStudent.initials}
					</span>
					<div>
						<h3 class="text-sm font-bold font-serif text-slate-900">{activeStudent.name}</h3>
						<p class="text-[9px] font-bold text-slate-400 uppercase tracking-widest mt-0.5">
							Enr No: {activeStudent.enrollment}
						</p>
					</div>
				</div>
				<button
					onclick={() => (isStudentModalOpen = false)}
					aria-label="Close modal"
					class="p-1 rounded-lg text-slate-400 hover:bg-slate-100 hover:text-slate-600 transition-colors"
				>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						fill="none"
						viewBox="0 0 24 24"
						stroke-width="2"
						stroke="currentColor"
						class="w-5 h-5"
					>
						<path stroke-linecap="round" stroke-linejoin="round" d="M6 18 18 6M6 6l12 12" />
					</svg>
				</button>
			</div>

			<div class="p-6 space-y-4">
				<div class="space-y-2 bg-slate-50 p-4 border border-slate-150 rounded-xl">
					<div>
						<span class="text-[9px] font-bold text-slate-400 uppercase tracking-wider block"
							>Enrolled Department</span
						>
						<span class="text-xs font-bold text-slate-800 block mt-0.5"
							>{activeStudent.courseName || activeStudent.course_name || 'IIPS Department'}</span
						>
					</div>
					<div class="pt-2 border-t border-slate-150">
						<span class="text-[9px] font-bold text-slate-400 uppercase tracking-wider block"
							>Current Extracurricular Credits</span
						>
						<span class="text-xs font-extrabold text-[#881B1B] block mt-0.5"
							>{activeStudent.creditsEarned ?? activeStudent.credits_earned ?? 0} Credits completed</span
						>
					</div>
				</div>

				<div class="space-y-1.5">
					<span class="text-[9px] font-bold text-slate-400 uppercase tracking-wider block"
						>Follow-up Reason / Issue</span
					>
					<div class="p-4 border border-rose-100 rounded-xl bg-rose-50/20 text-rose-900 space-y-1">
						<div class="text-xs font-bold uppercase tracking-wider flex items-center gap-1.5">
							<span class="w-1.5 h-1.5 bg-rose-600 rounded-full animate-pulse"></span>
							{activeStudent.issue}
						</div>
						<p class="text-[11px] leading-relaxed text-slate-600 pt-1 font-medium">
							The student has had this outstanding item for <span
								class="font-extrabold text-slate-800">{activeStudent.days} days</span
							>. Reminders should be dispatched immediately to avoid credit lockout periods.
						</p>
					</div>
				</div>
			</div>

			<div class="p-5 border-t border-slate-150 flex items-center justify-between bg-slate-50/30">
				<button
					onclick={() => handleSendReminder(activeStudent!)}
					class="px-4 py-2 border border-rose-200 bg-rose-50 hover:bg-rose-100 text-rose-600 font-extrabold text-xs uppercase tracking-wider rounded-lg transition-colors focus:outline-none"
				>
					Send Alert Reminder
				</button>
				<button
					onclick={() => (isStudentModalOpen = false)}
					class="px-4 py-2 bg-slate-100 hover:bg-slate-200 text-slate-700 font-bold text-xs uppercase rounded-lg transition-colors focus:outline-none"
				>
					Close Profile
				</button>
			</div>
		</div>
	</div>
{/if}
