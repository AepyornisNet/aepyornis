import { formatNumber } from '@angular/common';
import {
	ChangeDetectionStrategy,
	Component,
	computed,
	effect,
	inject,
	input,
	LOCALE_ID,
	signal,
} from '@angular/core';
import { firstValueFrom } from 'rxjs';
import { TranslatePipe } from '@ngx-translate/core';

import { Api } from '../../../../core/services/api';
import { WorkoutDetail, WorkoutRangeStats, WorkoutRangeStatsUnits } from '../../../../core/types/workout';
import { IntervalSelection, WorkoutDetailCoordinatorService } from '../../services/workout-detail-coordinator.service';

type NumericRangeStatKey = {
	[K in keyof WorkoutRangeStats]: WorkoutRangeStats[K] extends number | undefined ? K : never;
}[keyof WorkoutRangeStats];

type RangeStatConfig = {
	key: string;
	labelKey: string;
	unit: (units: WorkoutRangeStatsUnits) => string;
	decimals?: number;
	averageField?: NumericRangeStatKey;
	movingField?: NumericRangeStatKey;
	minField?: NumericRangeStatKey;
	maxField?: NumericRangeStatKey;
	metricKey?: string;
	ignoreZero?: boolean;
};

type WorkoutStatRow = {
	labelKey: string;
	value?: string;
};

type WorkoutStatCard = {
	key: string;
	labelKey: string;
	rows: WorkoutStatRow[];
};

const RANGE_CONFIGS: RangeStatConfig[] = [
	{
		key: 'speed',
		labelKey: 'shared.Speed',
		unit: (units) => units.speed,
		decimals: 1,
		averageField: 'average_speed',
		movingField: 'average_speed_no_pause',
		minField: 'min_speed',
		maxField: 'max_speed',
		ignoreZero: false,
	},
	{
		key: 'cadence',
		labelKey: 'shared.Cadence',
		unit: () => 'rpm',
		decimals: 0,
		averageField: 'average_cadence',
		minField: 'min_cadence',
		maxField: 'max_cadence',
		metricKey: 'cadence',
	},
	{
		key: 'heart-rate',
		labelKey: 'shared.Heart_rate',
		unit: () => 'bpm',
		decimals: 0,
		averageField: 'average_heart_rate',
		minField: 'min_heart_rate',
		maxField: 'max_heart_rate',
		metricKey: 'heart-rate',
	},
	{
		key: 'power',
		labelKey: 'shared.Power',
		unit: () => 'W',
		decimals: 0,
		averageField: 'average_power',
		minField: 'min_power',
		maxField: 'max_power',
		metricKey: 'power',
	},
	{
		key: 'slope',
		labelKey: 'shared.Slope',
		unit: () => '%',
		decimals: 1,
		averageField: 'average_slope',
		minField: 'min_slope',
		maxField: 'max_slope',
		ignoreZero: false,
	},
	{
		key: 'temperature',
		labelKey: 'shared.temperature',
		unit: (units) => units.temperature,
		decimals: 1,
		ignoreZero: false,
		averageField: 'average_temperature',
		minField: 'min_temperature',
		maxField: 'max_temperature',
		metricKey: 'temperature',
	},
];

@Component({
	selector: 'app-workout-statistics',
	imports: [TranslatePipe],
	templateUrl: './workout-statistics.html',
	styleUrl: './workout-statistics.scss',
	changeDetection: ChangeDetectionStrategy.OnPush,
})
export class WorkoutStatisticsComponent {
	public readonly workout = input<WorkoutDetail | null>(null);
	private readonly locale = inject(LOCALE_ID);
	private readonly api = inject(Api);
	private readonly coordinator = inject(WorkoutDetailCoordinatorService);

	private readonly stats = signal<WorkoutRangeStats | null>(null);
	private readonly loading = signal(false);
	private requestId = 0;

	public constructor() {
		effect(() => {
			const workout = this.workout();
			const selection = this.coordinator.selectedInterval();
			void this.loadStats(workout, selection);
		});
	}

	public readonly cards = computed<WorkoutStatCard[]>(() => {
		const workout = this.workout();
		const stats = this.stats();
		if (!workout || !stats) {
			return [];
		}

		const units = this.resolveUnits(stats);
		const selectionActive = Boolean(this.coordinator.selectedInterval());
		const availableMetrics = workout.map_data?.extra_metrics ?? [];

		const cards: WorkoutStatCard[] = [];

		const distanceCard = this.buildDistanceCard(
			stats,
			units,
			selectionActive,
			workout.laps?.length ?? 0,
		);
		if (distanceCard) {
			cards.push(distanceCard);
		}

		const elevationCard = this.buildElevationCard(stats, units);
		if (elevationCard) {
			cards.push(elevationCard);
		}

		RANGE_CONFIGS.forEach((config) => {
			const rangeCard = this.buildRangeCard(stats, config, units, availableMetrics);
			if (rangeCard) {
				cards.push(rangeCard);
			}
		});

		return cards;
	});

	public hasStatistics(): boolean {
		return this.cards().length > 0;
	}

	private async loadStats(workout: WorkoutDetail | null, selection: IntervalSelection | null): Promise<void> {
		if (!workout?.id || !workout.map_data?.details?.distance?.length) {
			this.stats.set(null);
			this.loading.set(false);
			return;
		}

		const params =
			selection && selection.startIndex >= 0 && selection.endIndex >= selection.startIndex
				? { start_index: selection.startIndex, end_index: selection.endIndex }
				: undefined;

		const currentRequest = ++this.requestId;
		this.loading.set(true);

		try {
			const response = await firstValueFrom(this.api.getWorkoutRangeStats(workout.id, params));
			if (this.requestId !== currentRequest) {
				return;
			}
			this.stats.set(response.results);
		} catch (error) {
			if (this.requestId === currentRequest) {
				console.error('Failed to load workout range stats', error);
				this.stats.set(null);
			}
		} finally {
			if (this.requestId === currentRequest) {
				this.loading.set(false);
			}
		}
	}

	private resolveUnits(stats: WorkoutRangeStats | null): WorkoutRangeStatsUnits {
		return stats?.units ?? { distance: 'km', speed: 'km/h', elevation: 'm', temperature: '°C' };
	}

	private buildDistanceCard(
		stats: WorkoutRangeStats,
		units: WorkoutRangeStatsUnits,
		selectionActive: boolean,
		lapCount: number,
	): WorkoutStatCard | null {
		const rows: WorkoutStatRow[] = [];

		if (stats.distance > 0) {
			rows.push({
				labelKey: selectionActive ? 'shared.Distance' : 'shared.Total_distance',
				value: this.formatDistance(stats.distance, units.distance),
			});
		}

		if (stats.duration > 0) {
			rows.push({ labelKey: 'shared.Duration', value: this.formatDurationValue(stats.duration) });
		}

		if (Number.isFinite(stats.pause_duration) && stats.pause_duration >= 0) {
			rows.push({ labelKey: 'workout.pause_time', value: this.formatDurationValue(stats.pause_duration) });
		}

		if (!selectionActive && lapCount > 0) {
			rows.push({ labelKey: 'workout.laps', value: lapCount.toString() });
		}

		return rows.length
			? {
					key: 'distance-summary',
					labelKey: 'shared.Distance',
					rows,
			  }
			: null;
	}

	private buildElevationCard(
		stats: WorkoutRangeStats,
		units: WorkoutRangeStatsUnits,
	): WorkoutStatCard | null {
		const rows: WorkoutStatRow[] = [];

		if (stats.total_up > 0) {
			rows.push({ labelKey: 'workout.elev_up', value: this.formatElevation(stats.total_up, units.elevation) });
		}

		if (stats.total_down > 0) {
			rows.push({ labelKey: 'workout.elev_down', value: this.formatElevation(stats.total_down, units.elevation) });
		}

		if (stats.max_elevation > stats.min_elevation) {
			const elevationRange = `${this.formatElevation(stats.min_elevation, units.elevation)} - ${this.formatElevation(
				stats.max_elevation,
				units.elevation,
			)}`;

			rows.push({
				labelKey: 'dashboard.elevation_range',
				value: elevationRange,
			});
		}

		return rows.length
			? {
					key: 'elevation-summary',
					labelKey: 'shared.Elevation',
					rows,
			  }
			: null;
	}

	private buildRangeCard(
		stats: WorkoutRangeStats,
		config: RangeStatConfig,
		units: WorkoutRangeStatsUnits,
		availableMetrics: string[],
	): WorkoutStatCard | null {
		if (config.metricKey && !availableMetrics.includes(config.metricKey)) {
			return null;
		}

		const rows: WorkoutStatRow[] = [];
		const average = this.resolveValue(stats, config.averageField, config.ignoreZero !== false);
		const moving = this.resolveValue(stats, config.movingField, config.ignoreZero !== false);
		const min = this.resolveValue(stats, config.minField, config.ignoreZero !== false);
		const max = this.resolveValue(stats, config.maxField, config.ignoreZero !== false);
		const unitLabel = config.unit(units);

		if (average !== undefined) {
			rows.push({ labelKey: 'shared.average', value: this.formatRangeValue(average, unitLabel, config.decimals) });
		}

		if (moving !== undefined && config.movingField) {
			rows.push({
				labelKey: 'dashboard.average_speed_no_pause',
				value: this.formatRangeValue(moving, unitLabel, config.decimals),
			});
		}

		if (min !== undefined) {
			rows.push({ labelKey: 'shared.minimum', value: this.formatRangeValue(min, unitLabel, config.decimals) });
		}

		if (max !== undefined) {
			rows.push({ labelKey: 'shared.maximum', value: this.formatRangeValue(max, unitLabel, config.decimals) });
		}

		return rows.length
			? {
					key: config.key,
					labelKey: config.labelKey,
					rows,
			  }
			: null;
	}

	private resolveValue(
		stats: WorkoutRangeStats,
		field: NumericRangeStatKey | undefined,
		ignoreZero: boolean,
	): number | undefined {
		if (!field) {
			return undefined;
		}

		const value = stats[field];
		if (typeof value !== 'number' || Number.isNaN(value)) {
			return undefined;
		}

		if (ignoreZero && value === 0) {
			return undefined;
		}

		return value;
	}

	private formatDistance(value: number, unit: string): string {
		const formatted = formatNumber(value, this.locale, '1.2-2');
		return `${formatted} ${unit}`;
	}

	private formatElevation(value: number, unit: string): string {
		const formatted = formatNumber(value, this.locale, '1.0-1');
		return `${formatted} ${unit}`;
	}

	private formatRangeValue(value: number, unit: string, decimals: number | undefined): string {
		if (value === undefined || Number.isNaN(value)) {
			return '-';
		}

		const digits = decimals !== undefined && decimals > 0 ? `1.${decimals}-${decimals}` : '1.0-0';
		const formatted = formatNumber(value, this.locale, digits);
		return unit ? `${formatted} ${unit}` : formatted;
	}

	private formatDurationValue(seconds: number): string {
		const totalSeconds = Math.round(seconds);
		const hours = Math.floor(totalSeconds / 3600);
		const minutes = Math.floor((totalSeconds % 3600) / 60);
		const secs = totalSeconds % 60;

		if (hours > 0) {
			return `${hours}h ${minutes}m ${secs}s`;
		}
		if (minutes > 0) {
			return `${minutes}m ${secs}s`;
		}
		return `${secs}s`;
	}
}

export function hasWorkoutStatistics(workout: WorkoutDetail | null): boolean {
	const details = workout?.map_data?.details;
	return Boolean(details && details.distance && details.distance.length > 1);
}
