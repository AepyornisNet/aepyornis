import { formatNumber } from '@angular/common';
import {
	ChangeDetectionStrategy,
	Component,
	computed,
	inject,
	input,
	LOCALE_ID,
} from '@angular/core';
import { TranslatePipe } from '@ngx-translate/core';

import { WorkoutDetail } from '../../../../core/types/workout';
import { IntervalSelection, WorkoutDetailCoordinatorService } from '../../services/workout-detail-coordinator.service';

type NumericWorkoutKeys = {
	[K in keyof WorkoutDetail]: WorkoutDetail[K] extends number | undefined ? K : never;
}[keyof WorkoutDetail];

type SeriesSource = 'speed' | 'slope' | 'extraMetric';

type SelectionBounds = {
	startIndex: number;
	endIndex: number;
};

type RangeStatConfig = {
	key: string;
	labelKey: string;
	unit?: string;
	decimals?: number;
	averageField?: NumericWorkoutKeys;
	minField?: NumericWorkoutKeys;
	maxField?: NumericWorkoutKeys;
	seriesSource?: SeriesSource;
	seriesKey?: string;
	seriesTransform?: (value: number) => number;
	fieldTransform?: (value: number) => number;
	ignoreZero?: boolean;
};

type WorkoutStatNumbers = {
	average?: number;
	min?: number;
	max?: number;
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

type ElevationStats = {
	up?: number;
	down?: number;
	min?: number;
	max?: number;
};

const MOVING_SPEED_THRESHOLD = 0.5; // m/s
const metersPerSecondToKilometersPerHour = (value: number): number => value * 3.6;

const RANGE_CONFIGS: RangeStatConfig[] = [
	{
		key: 'speed',
		labelKey: 'shared.Speed',
		unit: 'km/h',
		decimals: 1,
		averageField: 'average_speed',
		maxField: 'max_speed',
		seriesSource: 'speed',
		seriesTransform: metersPerSecondToKilometersPerHour,
		fieldTransform: metersPerSecondToKilometersPerHour,
	},
	{
		key: 'cadence',
		labelKey: 'shared.Cadence',
		unit: 'rpm',
		decimals: 0,
		averageField: 'average_cadence',
		maxField: 'max_cadence',
		seriesSource: 'extraMetric',
		seriesKey: 'cadence',
	},
	{
		key: 'heart-rate',
		labelKey: 'shared.Heart_rate',
		unit: 'bpm',
		decimals: 0,
		averageField: 'average_heart_rate',
		maxField: 'max_heart_rate',
		seriesSource: 'extraMetric',
		seriesKey: 'heart-rate',
	},
	{
		key: 'power',
		labelKey: 'shared.Power',
		unit: 'W',
		decimals: 0,
		averageField: 'average_power',
		maxField: 'max_power',
		seriesSource: 'extraMetric',
		seriesKey: 'power',
	},
	{
		key: 'slope',
		labelKey: 'shared.Slope',
		unit: '%',
		decimals: 1,
		seriesSource: 'slope',
		ignoreZero: false,
	},
	{
		key: 'temperature',
		labelKey: 'shared.temperature',
		unit: '\u00B0C',
		decimals: 1,
		seriesSource: 'extraMetric',
		seriesKey: 'temperature',
		ignoreZero: false,
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
	private readonly coordinator = inject(WorkoutDetailCoordinatorService);

	public readonly cards = computed<WorkoutStatCard[]>(() => {
		const workout = this.workout();
		if (!workout) {
			return [];
		}

		const selection = this.coordinator.selectedInterval();
		const bounds = determineSelectionBounds(workout, selection);
		const selectionActive = Boolean(selection && bounds);

		const cards: WorkoutStatCard[] = [];

		const distanceCard = buildDistanceCard(workout, bounds, selectionActive, this.locale);
		if (distanceCard) {
			cards.push(distanceCard);
		}

		const elevationCard = buildElevationCard(workout, bounds, this.locale);
		if (elevationCard) {
			cards.push(elevationCard);
		}

		RANGE_CONFIGS.forEach((config) => {
			const rangeCard = buildRangeCard(workout, config, bounds, this.locale);
			if (rangeCard) {
				cards.push(rangeCard);
			}
		});

		return cards;
	});

	public hasStatistics(): boolean {
		return this.cards().length > 0;
	}
}

export function hasWorkoutStatistics(workout: WorkoutDetail | null): boolean {
	if (!workout) {
		return false;
	}

	if (hasDistanceData(workout) || hasElevationData(workout)) {
		return true;
	}

	return RANGE_CONFIGS.some((config) => Boolean(computeStatNumbers(workout, config, null)));
}

function buildDistanceCard(
	workout: WorkoutDetail,
	bounds: SelectionBounds | null,
	selectionActive: boolean,
	locale: string,
): WorkoutStatCard | null {
	const rows: WorkoutStatRow[] = [];
	const distanceMeters = calculateDistanceMeters(workout, bounds);
	if (distanceMeters !== undefined) {
		rows.push({
			labelKey: selectionActive ? 'shared.Distance' : 'shared.Total_distance',
			value: formatDistance(distanceMeters, locale),
		});
	}

	const durationSeconds = calculateDurationSeconds(workout, bounds);
	if (durationSeconds !== undefined) {
		rows.push({ labelKey: 'shared.Duration', value: formatDurationValue(durationSeconds) });
	}

	if (!selectionActive) {
		const lapCount = workout.laps?.length ?? 0;
		if (lapCount > 0) {
			rows.push({ labelKey: 'workout.laps', value: lapCount.toString() });
		}
	}

	return rows.length > 0
		? {
				key: 'distance-summary',
				labelKey: 'shared.Distance',
				rows,
			}
		: null;
}

function buildElevationCard(
	workout: WorkoutDetail,
	bounds: SelectionBounds | null,
	locale: string,
): WorkoutStatCard | null {
	const rows: WorkoutStatRow[] = [];
	const stats = calculateElevationStats(workout, bounds);

	if (stats?.up && stats.up > 0) {
		rows.push({ labelKey: 'workout.elev_up', value: formatElevation(stats.up, locale) });
	}

	if (stats?.down && stats.down > 0) {
		rows.push({ labelKey: 'workout.elev_down', value: formatElevation(stats.down, locale) });
	}

	if (stats?.min !== undefined && stats?.max !== undefined && stats.max > stats.min) {
		rows.push({
			labelKey: 'dashboard.elevation_range',
			value: `${formatElevation(stats.min, locale)} - ${formatElevation(stats.max, locale)}`,
		});
	}

	return rows.length > 0
		? {
				key: 'elevation-summary',
				labelKey: 'shared.Elevation',
				rows,
			}
		: null;
}

function buildRangeCard(
	workout: WorkoutDetail,
	config: RangeStatConfig,
	bounds: SelectionBounds | null,
	locale: string,
): WorkoutStatCard | null {
	const numbers = computeStatNumbers(workout, config, bounds);
	if (!numbers) {
		return null;
	}

	const rows: WorkoutStatRow[] = [];
	const movingAverage =
		config.key === 'speed' ? calculateMovingAverageSpeed(workout, bounds) : undefined;
	if (numbers.average !== undefined) {
		rows.push({
			labelKey: 'shared.average',
			value: formatRangeValue(numbers.average, config, locale),
		});
	}
	if (movingAverage !== undefined) {
		rows.push({
			labelKey: 'dashboard.average_speed_no_pause',
			value: formatRangeValue(movingAverage, config, locale),
		});
	}
	if (numbers.min !== undefined) {
		rows.push({
			labelKey: 'shared.minimum',
			value: formatRangeValue(numbers.min, config, locale),
		});
	}
	if (numbers.max !== undefined) {
		rows.push({
			labelKey: 'shared.maximum',
			value: formatRangeValue(numbers.max, config, locale),
		});
	}

	return rows.length > 0
		? {
				key: config.key,
				labelKey: config.labelKey,
				rows,
			}
		: null;
}

function computeStatNumbers(
	workout: WorkoutDetail,
	config: RangeStatConfig,
	bounds: SelectionBounds | null,
): WorkoutStatNumbers | null {
	const ignoreZero = config.ignoreZero !== false;
	const seriesStats = computeSeriesStats(workout, config, ignoreZero, bounds);

	const averageValue =
		getFieldValue(workout, config.averageField, config.fieldTransform, ignoreZero) ??
		seriesStats.average;
	const minValue =
		getFieldValue(workout, config.minField, config.fieldTransform, ignoreZero) ??
		seriesStats.min;
	const maxValue =
		getFieldValue(workout, config.maxField, config.fieldTransform, ignoreZero) ??
		seriesStats.max;

	if (averageValue === undefined && minValue === undefined && maxValue === undefined) {
		return null;
	}

	return {
		average: averageValue,
		min: minValue,
		max: maxValue,
	};
}

function getFieldValue(
	workout: WorkoutDetail,
	field: NumericWorkoutKeys | undefined,
	transform: ((value: number) => number) | undefined,
	ignoreZero: boolean,
): number | undefined {
	if (!field) {
		return undefined;
	}

	const rawValue = workout[field];
	if (typeof rawValue !== 'number' || Number.isNaN(rawValue)) {
		return undefined;
	}

	const transformed = transform ? transform(rawValue) : rawValue;
	if (Number.isNaN(transformed)) {
		return undefined;
	}

	if (ignoreZero && transformed <= 0) {
		return undefined;
	}

	return transformed;
}

function computeSeriesStats(
	workout: WorkoutDetail,
	config: RangeStatConfig,
	ignoreZero: boolean,
	bounds: SelectionBounds | null,
): WorkoutStatNumbers {
	const values = resolveSeriesValues(workout, config);
	if (!values || values.length === 0) {
		return {};
	}

	const sliced = sliceValues(values, bounds);
	if (!sliced.length) {
		return {};
	}

	const transformed = sliced
		.map((value) => {
			if (value === null || value === undefined) {
				return undefined;
			}
			const result = config.seriesTransform ? config.seriesTransform(value) : value;
			if (Number.isNaN(result)) {
				return undefined;
			}
			return result;
		})
		.filter((value): value is number => value !== undefined && (!ignoreZero || value > 0));

	if (transformed.length === 0) {
		return {};
	}

	const sum = transformed.reduce((total, value) => total + value, 0);
	return {
		average: sum / transformed.length,
		min: Math.min(...transformed),
		max: Math.max(...transformed),
	};
}

function resolveSeriesValues(
	workout: WorkoutDetail,
	config: RangeStatConfig,
): (number | null | undefined)[] | undefined {
	const details = workout.map_data?.details;
	if (!details) {
		return undefined;
	}

	switch (config.seriesSource) {
		case 'speed':
			return details.speed;
		case 'slope':
			return details.slope;
		case 'extraMetric':
			if (!config.seriesKey) {
				return undefined;
			}
			return details.extra_metrics?.[config.seriesKey];
		default:
			return undefined;
	}
}

function sliceValues(
	values: (number | null | undefined)[],
	bounds: SelectionBounds | null,
): (number | null | undefined)[] {
	if (!bounds) {
		return values;
	}
	if (values.length === 0) {
		return values;
	}

	const startIndex = Math.min(Math.max(bounds.startIndex, 0), Math.max(values.length - 1, 0));
	const endIndex = Math.min(Math.max(bounds.endIndex, startIndex), values.length - 1);
	return values.slice(startIndex, endIndex + 1);
}

function determineSelectionBounds(
	workout: WorkoutDetail,
	selection: IntervalSelection | null,
): SelectionBounds | null {
	const details = workout.map_data?.details;
	const length = Math.max(
		details?.distance?.length ?? 0,
		details?.duration?.length ?? 0,
		details?.elevation?.length ?? 0,
	);
	if (!details || !length || length < 2) {
		return null;
	}

	let startIndex = 0;
	let endIndex = length - 1;

	if (selection) {
		startIndex = clamp(selection.startIndex, 0, length - 2);
		endIndex = clamp(selection.endIndex, startIndex + 1, length - 1);
	}

	return { startIndex, endIndex };
}

function calculateDistanceMeters(
	workout: WorkoutDetail,
	bounds: SelectionBounds | null,
): number | undefined {
	if (bounds && workout.map_data?.details?.distance) {
		const distances = workout.map_data.details.distance;
		if (distances.length >= 2) {
			const startIndex = Math.min(Math.max(bounds.startIndex, 0), Math.max(distances.length - 2, 0));
			const endIndex = Math.min(Math.max(bounds.endIndex, startIndex + 1), distances.length - 1);
			const start = distances[startIndex];
			const end = distances[endIndex];
			if (typeof start === 'number' && typeof end === 'number') {
				return Math.max(0, (end - start) * 1000);
			}
		}
	}

	return resolveTotalDistance(workout);
}

function calculateDurationSeconds(
	workout: WorkoutDetail,
	bounds: SelectionBounds | null,
): number | undefined {
	if (bounds && workout.map_data?.details?.duration) {
		const durations = workout.map_data.details.duration;
		if (durations.length >= 2) {
			const startIndex = Math.min(Math.max(bounds.startIndex, 0), Math.max(durations.length - 2, 0));
			const endIndex = Math.min(Math.max(bounds.endIndex, startIndex + 1), durations.length - 1);
			const start = durations[startIndex];
			const end = durations[endIndex];
			if (typeof start === 'number' && typeof end === 'number') {
				return Math.max(0, end - start);
			}
		}
	}

	return typeof workout.total_duration === 'number' ? workout.total_duration : undefined;
}

function calculateMovingAverageSpeed(
	workout: WorkoutDetail,
	bounds: SelectionBounds | null,
): number | undefined {
	const distanceMeters = calculateDistanceMeters(workout, bounds);
	const movingDurationSeconds = calculateMovingDurationSeconds(workout, bounds);

	if (distanceMeters !== undefined && movingDurationSeconds && movingDurationSeconds > 0) {
		return metersPerSecondToKilometersPerHour(distanceMeters / movingDurationSeconds);
	}

	if (!bounds && typeof workout.average_speed_no_pause === 'number' && workout.average_speed_no_pause > 0) {
		return metersPerSecondToKilometersPerHour(workout.average_speed_no_pause);
	}

	return undefined;
}

function calculateMovingDurationSeconds(
	workout: WorkoutDetail,
	bounds: SelectionBounds | null,
): number | undefined {
	const details = workout.map_data?.details;
	const durations = details?.duration;
	const speeds = details?.speed;

	let effectiveBounds = bounds;
	if (!effectiveBounds && details) {
		effectiveBounds = determineSelectionBounds(workout, null);
	}

	if (effectiveBounds && durations && speeds) {
		let movingSeconds = 0;
		const length = Math.min(durations.length, speeds.length);
		for (let i = Math.max(effectiveBounds.startIndex + 1, 1); i <= effectiveBounds.endIndex && i < length; i++) {
			const delta = durations[i] - durations[i - 1];
			const speed = speeds[i];
			if (delta <= 0 || speed === null || speed === undefined) {
				continue;
			}
			if (speed > MOVING_SPEED_THRESHOLD) {
				movingSeconds += delta;
			}
		}
		if (movingSeconds > 0) {
			return movingSeconds;
		}
	}

	if (!bounds && typeof workout.total_duration === 'number') {
		if (typeof workout.pause_duration === 'number' && workout.pause_duration > 0) {
			return Math.max(0, workout.total_duration - workout.pause_duration);
		}
		return workout.total_duration;
	}

	return undefined;
}

function calculateElevationStats(
	workout: WorkoutDetail,
	bounds: SelectionBounds | null,
): ElevationStats | null {
	if (bounds) {
		const selectionStats = calculateSelectionElevation(workout, bounds);
		if (selectionStats) {
			return selectionStats;
		}
	}

	return {
		up: workout.total_up,
		down: workout.total_down,
		min: workout.min_elevation,
		max: workout.max_elevation,
	};
}

function calculateSelectionElevation(
	workout: WorkoutDetail,
	bounds: SelectionBounds,
): ElevationStats | null {
	const elevations = workout.map_data?.details?.elevation;
	if (!elevations || elevations.length < 2) {
		return null;
	}

	const startIndex = Math.min(Math.max(bounds.startIndex, 0), Math.max(elevations.length - 2, 0));
	const endIndex = Math.min(Math.max(bounds.endIndex, startIndex + 1), elevations.length - 1);

	let min = Infinity;
	let max = -Infinity;
	let up = 0;
	let down = 0;

	for (let i = startIndex + 1; i <= endIndex; i++) {
		const current = elevations[i];
		const previous = elevations[i - 1];
		if (current === undefined || previous === undefined) {
			continue;
		}

		min = Math.min(min, current, previous);
		max = Math.max(max, current, previous);

		const delta = current - previous;
		if (delta > 0) {
			up += delta;
		} else {
			down += Math.abs(delta);
		}
	}

	if (!Number.isFinite(min) || !Number.isFinite(max)) {
		return null;
	}

	return {
		up: up > 0 ? up : undefined,
		down: down > 0 ? down : undefined,
		min,
		max,
	};
}

function hasDistanceData(workout: WorkoutDetail): boolean {
	return resolveTotalDistance(workout) !== undefined || (workout.laps?.length ?? 0) > 0;
}

function hasElevationData(workout: WorkoutDetail): boolean {
	return (
		(typeof workout.total_up === 'number' && workout.total_up > 0) ||
		(typeof workout.total_down === 'number' && workout.total_down > 0) ||
		(typeof workout.min_elevation === 'number' && typeof workout.max_elevation === 'number' &&
			workout.max_elevation > workout.min_elevation)
	);
}

function resolveTotalDistance(workout: WorkoutDetail): number | undefined {
	if (typeof workout.total_distance === 'number' && workout.total_distance > 0) {
		return workout.total_distance;
	}

	if (workout.map_data?.details?.distance?.length) {
		const lastDistance = workout.map_data.details.distance[workout.map_data.details.distance.length - 1];
		if (typeof lastDistance === 'number' && lastDistance > 0) {
			return lastDistance * 1000;
		}
	}

	if (workout.laps?.length) {
		const total = workout.laps.reduce((sum, lap) => sum + (lap.total_distance ?? 0), 0);
		return total > 0 ? total : undefined;
	}

	return undefined;
}

function formatDistance(valueMeters: number, locale: string): string {
	const kilometers = valueMeters / 1000;
	const formatted = formatNumber(kilometers, locale, '1.2-2');
	return `${formatted} km`;
}

function formatElevation(valueMeters: number, locale: string): string {
	const formatted = formatNumber(valueMeters, locale, '1.0-0');
	return `${formatted} m`;
}

function formatRangeValue(value: number, config: RangeStatConfig, locale: string): string {
	if (value === undefined || Number.isNaN(value)) {
		return '-';
	}

	const decimals = config.decimals ?? 0;
	const digits = decimals > 0 ? `1.${decimals}-${decimals}` : '1.0-0';
	const formatted = formatNumber(value, locale, digits);
	return config.unit ? `${formatted} ${config.unit}` : formatted;
}

function formatDurationValue(seconds: number): string {
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

function clamp(value: number, min: number, max: number): number {
	return Math.min(Math.max(value, min), max);
}
