import {
  AfterViewInit,
  ApplicationRef,
  ChangeDetectionStrategy,
  Component,
  createComponent,
  ElementRef,
  EnvironmentInjector,
  inject,
  OnDestroy,
  signal,
  viewChild,
} from '@angular/core';

import { FormsModule } from '@angular/forms';
import { TranslatePipe } from '@ngx-translate/core';
import { firstValueFrom } from 'rxjs';
import * as L from 'leaflet';
import 'leaflet.heat';
import 'leaflet.markercluster';
import { Api } from '../../../../core/services/api';
import { GeoJsonFeatureCollection, WorkoutPopupData } from '../../../../core/types/statistics';
import { WorkoutPopup } from '../../components/workout-popup/workout-popup';

// Fix for default marker icons
// eslint-disable-next-line @typescript-eslint/no-explicit-any
delete (L.Icon.Default.prototype as any)._getIconUrl;
L.Icon.Default.mergeOptions({
  iconRetinaUrl: '/leaflet/images/marker-icon-2x.png',
  iconUrl: '/leaflet/images/marker-icon.png',
  shadowUrl: '/leaflet/images/marker-shadow.png',
});

@Component({
  selector: 'app-heatmap',
  imports: [FormsModule, TranslatePipe],
  templateUrl: './heatmap.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
  styleUrls: ['./heatmap.scss'],
})
export class Heatmap implements AfterViewInit, OnDestroy {
  private readonly mapContainer = viewChild<ElementRef<HTMLDivElement>>('mapContainer');

  private api = inject(Api);
  private environmentInjector = inject(EnvironmentInjector);
  private applicationRef = inject(ApplicationRef);
  private map?: L.Map;
  private heatLayer?: L.HeatLayer;
  private markersLayer?: L.MarkerClusterGroup;

  public readonly loading = signal(true);
  public readonly error = signal<string | null>(null);

  // Control settings
  public readonly radius = signal(10);
  public readonly blur = signal(15);
  public readonly showMarkers = signal(true);
  public readonly onlyTrace = signal(false);

  private heatMapData: L.HeatLatLngTuple[] = [];

  public ngAfterViewInit(): void {
    this.initMap();
    this.loadHeatmapData();
  }

  public ngOnDestroy(): void {
    if (this.map) {
      this.map.remove();
    }
  }

  private initMap(): void {
    const containerRef = this.mapContainer();
    if (!containerRef) {
      return;
    }

    this.map = L.map(containerRef.nativeElement, {
      fadeAnimation: false,
    });

    const layerStreet = L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
      attribution: '&copy; <a href="http://www.openstreetmap.org/copyright">OpenStreetMap</a>',
      className: 'map-tiles',
    });

    const layerAerial = L.tileLayer(
      'https://server.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile/{z}/{y}/{x}',
      {
        attribution: 'Powered by Esri',
      },
    );

    L.control
      .layers({
        Streets: layerStreet,
        Aerial: layerAerial,
      })
      .addTo(this.map);

    layerStreet.addTo(this.map);

    // Add custom control for heatmap settings
    this.addCustomControl();
  }

  private addCustomControl(): void {
    if (!this.map) {
      return;
    }

    const CustomControl = L.Control.extend({
      options: { position: 'topright' },
      onAdd: () => {
        const container = L.DomUtil.create('div', 'leaflet-bar leaflet-control');
        container.style.backgroundColor = 'white';
        container.style.padding = '10px';
        container.innerHTML = `
          <div style="display: flex; flex-direction: column; gap: 8px;">
            <div style="display: flex; align-items: center;">
              <label for="radius" style="width: 80px; color: #333;">Radius</label>
              <input type="range" id="radius" value="${this.radius()}" min="1" max="30" style="padding: 0;"/>
            </div>
            <div style="display: flex; align-items: center;">
              <label for="blur" style="width: 80px; color: #333;">Blur</label>
              <input type="range" id="blur" value="${this.blur()}" min="1" max="30" style="padding: 0;"/>
            </div>
            <div style="display: flex; align-items: center;">
              <input type="checkbox" id="showMarkers" ${this.showMarkers() ? 'checked' : ''} style="margin-right: 4px;"/>
              <label for="showMarkers" style="color: #333;">Show Markers</label>
            </div>
            <div style="display: flex; align-items: center;">
              <input type="checkbox" id="onlyTrace" ${this.onlyTrace() ? 'checked' : ''} style="margin-right: 4px;"/>
              <label for="onlyTrace" style="color: #333;">Only show trace</label>
            </div>
          </div>
        `;

        L.DomEvent.disableClickPropagation(container);

        const radiusInput = container.querySelector('#radius') as HTMLInputElement;
        const blurInput = container.querySelector('#blur') as HTMLInputElement;
        const showMarkersInput = container.querySelector('#showMarkers') as HTMLInputElement;
        const onlyTraceInput = container.querySelector('#onlyTrace') as HTMLInputElement;

        radiusInput?.addEventListener('input', () => {
          this.radius.set(Number(radiusInput.value));
          this.rerenderHeatMap();
        });

        blurInput?.addEventListener('input', () => {
          this.blur.set(Number(blurInput.value));
          this.rerenderHeatMap();
        });

        showMarkersInput?.addEventListener('change', () => {
          this.showMarkers.set(showMarkersInput.checked);
          this.rerenderHeatMap();
        });

        onlyTraceInput?.addEventListener('change', () => {
          this.onlyTrace.set(onlyTraceInput.checked);
          this.rerenderHeatMap();
        });

        return container;
      },
    });

    new CustomControl().addTo(this.map);
  }

  private async loadHeatmapData(): Promise<void> {
    this.loading.set(true);
    this.error.set(null);

    try {
      const [coordinatesResponse, centersResponse] = await Promise.all([
        firstValueFrom(this.api.getWorkoutsCoordinates()),
        firstValueFrom(this.api.getWorkoutsCenters()),
      ]);

      if (coordinatesResponse?.results) {
        this.heatMapData = this.geoJson2heat(coordinatesResponse.results);
      }

      if (centersResponse?.results && this.map) {
        this.markersLayer = L.markerClusterGroup({ showCoverageOnHover: false });

        const geoJsonLayer = L.geoJSON(
          centersResponse.results as unknown as GeoJSON.GeoJsonObject,
          {
            onEachFeature: (feature: unknown, layer) => {
              const feat = feature as { properties?: Record<string, unknown> } | null;
              if (feat?.properties && feat.properties['popup_data']) {
                const popupData = feat.properties['popup_data'] as WorkoutPopupData;

                // Create Angular component for popup
                const componentRef = createComponent(WorkoutPopup, {
                  environmentInjector: this.environmentInjector,
                });

                // Set input data
                componentRef.setInput('data', popupData);

                // Attach to application
                this.applicationRef.attachView(componentRef.hostView);

                // Get the DOM element
                const popupElement = componentRef.location.nativeElement as HTMLElement;

                // Bind popup with the component's DOM
                layer.bindPopup(popupElement);

                // Clean up component when popup is closed
                layer.on('popupclose', () => {
                  this.applicationRef.detachView(componentRef.hostView);
                  componentRef.destroy();
                });
              }
            },
          },
        );

        this.markersLayer.addLayer(geoJsonLayer);

        if (this.showMarkers()) {
          this.markersLayer.addTo(this.map);
        }

        // Fit map to bounds
        if (this.markersLayer.getBounds().isValid()) {
          this.map.fitBounds(this.markersLayer.getBounds());
        }
      }

      this.rerenderHeatMap();
    } catch (err) {
      console.error('Failed to load heatmap data:', err);
      this.error.set('Failed to load heatmap data. Please try again.');
    } finally {
      this.loading.set(false);
    }
  }

  private geoJson2heat(geojson: GeoJsonFeatureCollection): L.HeatLatLngTuple[] {
    return geojson.features.map(
      (feature) =>
        [feature.geometry.coordinates[1], feature.geometry.coordinates[0], 1] as L.HeatLatLngTuple,
    );
  }

  private rerenderHeatMap(): void {
    if (!this.map || this.heatMapData.length === 0) {
      return;
    }

    // Remove existing heat layer
    if (this.heatLayer) {
      this.map.removeLayer(this.heatLayer);
    }

    // Configure heatmap based on settings
    const config: L.HeatMapOptions = {
      radius: this.onlyTrace() ? 1 : this.radius(),
      blur: this.onlyTrace() ? 1 : this.blur(),
    };

    if (this.onlyTrace()) {
      config.minOpacity = 1;
      config.gradient = { 0: 'blue' };
    }

    this.heatLayer = L.heatLayer(this.heatMapData, config);
    this.heatLayer?.addTo(this.map);

    // Toggle markers
    if (this.markersLayer) {
      if (this.showMarkers()) {
        this.markersLayer.addTo(this.map);
      } else {
        this.map.removeLayer(this.markersLayer);
      }
    }
  }
}
