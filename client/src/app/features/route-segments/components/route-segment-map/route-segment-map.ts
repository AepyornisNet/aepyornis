import { ChangeDetectionStrategy, Component, effect, input } from '@angular/core';

import { NgxMapLibreGLModule } from '@maplibre/ngx-maplibre-gl';
import { LngLatBounds, Map, Marker } from 'maplibre-gl';
import { MapPoint } from '../../../../core/types/route-segment';
import { BaseMapComponent } from '../../../../core/components/base-map/base-map';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { TranslatePipe } from '@ngx-translate/core';

@Component({
  selector: 'app-route-segment-map',
  imports: [NgxMapLibreGLModule, AppIcon, TranslatePipe],
  templateUrl: './route-segment-map.html',
  styleUrls: ['./route-segment-map.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class RouteSegmentMapComponent extends BaseMapComponent {
  public readonly points = input<MapPoint[] | null>(null);
  public readonly selection = input<{ startIndex: number; endIndex: number } | null>(null);
  public readonly center = input<{ lat: number; lng: number } | null>(null);
  public readonly showControls = input<boolean>(false);

  private startMarker?: Marker;
  private endMarker?: Marker;
  private hasInitialFit = false;

  public constructor() {
    super();
    effect(() => {
      this.points();
      if (this.map && this.map.isStyleLoaded()) {
        this.renderTrack();
      }
    });

    effect(() => {
      this.highlightSelection(this.selection());
    });
  }

  public onMapLoad(map: Map): void {
    this.onMapLoadBase(map);
    this.refreshTrackAfterStyleChange();
  }

  private renderTrack(): void {
    const pts = this.points();
    if (!this.map || !pts || pts.length < 2) {
      return;
    }

    this.clearTrack();

    const coordinates = pts.map((p) => [p.lng, p.lat] as [number, number]);

    this.map.addSource('route-track-source', {
      type: 'geojson',
      data: {
        type: 'FeatureCollection',
        features: [
          {
            type: 'Feature',
            geometry: { type: 'LineString', coordinates },
            properties: {},
          },
        ],
      },
    });

    this.map.addLayer({
      id: 'route-track-layer',
      type: 'line',
      source: 'route-track-source',
      paint: {
        'line-color': 'green',
        'line-width': ['interpolate', ['linear'], ['zoom'], 0, 2, 10, 3, 15, 4],
        'line-opacity': 0.9,
      },
    });

    const sel = this.selection();
    const startIdx = sel ? Math.max(0, Math.min(sel.startIndex, pts.length - 1)) : 0;
    const endIdx = sel ? Math.max(0, Math.min(sel.endIndex, pts.length - 1)) : pts.length - 1;

    this.startMarker = new Marker({ color: '#198754', scale: 0.85 })
      .setLngLat([pts[startIdx].lng, pts[startIdx].lat])
      .addTo(this.map);

    this.endMarker = new Marker({ color: '#dc3545', scale: 0.85 })
      .setLngLat([pts[endIdx].lng, pts[endIdx].lat])
      .addTo(this.map);

    if (!this.hasInitialFit) {
      this.fitToCoordinates(coordinates, false);
      this.hasInitialFit = true;
    }

    this.highlightSelection(this.selection());
  }

  private highlightSelection(sel: { startIndex: number; endIndex: number } | null): void {
    if (!this.map || !this.points()) {
      return;
    }

    if (this.map.getLayer('route-highlight-layer')) {
      this.map.removeLayer('route-highlight-layer');
    }
    if (this.map.getSource('route-highlight-source')) {
      this.map.removeSource('route-highlight-source');
    }

    const pts = this.points()!;
    if (pts.length < 2) {
      return;
    }

    if (!sel) {
      if (this.startMarker) {
        this.startMarker.setLngLat([pts[0].lng, pts[0].lat]);
      }
      if (this.endMarker) {
        this.endMarker.setLngLat([pts[pts.length - 1].lng, pts[pts.length - 1].lat]);
      }
      return;
    }

    const start = Math.max(0, Math.min(sel.startIndex, pts.length - 2));
    const end = Math.max(start + 1, Math.min(sel.endIndex, pts.length - 1));
    const coords = pts.slice(start, end + 1).map((p) => [p.lng, p.lat] as [number, number]);

    this.map.addSource('route-highlight-source', {
      type: 'geojson',
      data: {
        type: 'FeatureCollection',
        features: [
          {
            type: 'Feature',
            geometry: { type: 'LineString', coordinates: coords },
            properties: {},
          },
        ],
      },
    });

    this.map.addLayer({
      id: 'route-highlight-layer',
      type: 'line',
      source: 'route-highlight-source',
      paint: {
        'line-color': '#0d6efd',
        'line-width': ['interpolate', ['linear'], ['zoom'], 0, 3, 10, 5, 15, 6],
        'line-opacity': 0.85,
      },
    });

    // Update marker positions smoothly without moving the camera
    if (this.startMarker && pts[start]) {
      this.startMarker.setLngLat([pts[start].lng, pts[start].lat]);
    }
    if (this.endMarker && pts[end]) {
      this.endMarker.setLngLat([pts[end].lng, pts[end].lat]);
    }
  }

  public fitToSelection(animate = true): void {
    const pts = this.points();
    const sel = this.selection();
    if (!this.map || !pts || pts.length === 0) {
      return;
    }

    if (!sel) {
      this.fitToRoute(animate);
      return;
    }

    const start = Math.max(0, Math.min(sel.startIndex, pts.length - 2));
    const end = Math.max(start + 1, Math.min(sel.endIndex, pts.length - 1));
    const coords = pts.slice(start, end + 1).map((p) => [p.lng, p.lat] as [number, number]);
    this.fitToCoordinates(coords, animate);
  }

  public fitToRoute(animate = true): void {
    const pts = this.points();
    if (!this.map || !pts || pts.length === 0) {
      return;
    }
    const coords = pts.map((p) => [p.lng, p.lat] as [number, number]);
    this.fitToCoordinates(coords, animate);
  }

  public focusStart(): void {
    const pts = this.points();
    const sel = this.selection();
    if (!this.map || !pts || pts.length === 0) {
      return;
    }
    const idx = sel ? Math.max(0, Math.min(sel.startIndex, pts.length - 1)) : 0;
    const pt = pts[idx];
    this.map.flyTo({
      center: [pt.lng, pt.lat],
      zoom: Math.max(this.map.getZoom(), 16),
      essential: true,
    });
  }

  public focusEnd(): void {
    const pts = this.points();
    const sel = this.selection();
    if (!this.map || !pts || pts.length === 0) {
      return;
    }
    const idx = sel ? Math.max(0, Math.min(sel.endIndex, pts.length - 1)) : pts.length - 1;
    const pt = pts[idx];
    this.map.flyTo({
      center: [pt.lng, pt.lat],
      zoom: Math.max(this.map.getZoom(), 16),
      essential: true,
    });
  }

  protected refreshAfterStyleChange(): void {
    this.refreshTrackAfterStyleChange();
  }

  private refreshTrackAfterStyleChange(): void {
    if (!this.map) {
      return;
    }

    const version = ++this.styleRefreshVersion;

    const tryRender = (): void => {
      if (!this.map || version !== this.styleRefreshVersion) {
        return;
      }
      if (this.map.isStyleLoaded()) {
        this.renderTrack();
        return;
      }
      window.setTimeout(tryRender, 50);
    };

    window.setTimeout(tryRender, 0);
  }

  private fitToCoordinates(coords: [number, number][], animate = false): void {
    if (!this.map || coords.length === 0) {
      return;
    }

    const bounds = new LngLatBounds(coords[0], coords[0]);
    for (const coord of coords) {
      bounds.extend(coord);
    }

    this.map.fitBounds(bounds, {
      padding: 60,
      animate,
      duration: animate ? 700 : 0,
    });
  }

  private clearTrack(): void {
    if (!this.map) {
      return;
    }

    const layers = ['route-highlight-layer', 'route-track-layer'];
    const sources = ['route-highlight-source', 'route-track-source'];

    for (const layer of layers) {
      if (this.map.getLayer(layer)) {
        this.map.removeLayer(layer);
      }
    }

    for (const source of sources) {
      if (this.map.getSource(source)) {
        this.map.removeSource(source);
      }
    }

    this.startMarker?.remove();
    this.endMarker?.remove();
  }
}
