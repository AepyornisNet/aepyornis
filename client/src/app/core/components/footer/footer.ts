import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { AppConfig } from '../../../core/services/app-config';
import { TranslatePipe } from '@ngx-translate/core';
import { RouterLink } from '@angular/router';

@Component({
  selector: 'app-footer',
  imports: [TranslatePipe, RouterLink],
  templateUrl: './footer.html',
  styleUrl: './footer.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Footer {
  private appConfig = inject(AppConfig);

  public readonly version = computed(() => this.appConfig.getVersion());
  public readonly versionSha = computed(() => this.appConfig.getVersionSha());
  public readonly hasLegalNotice = computed(
    () => this.appConfig.getLegalNoticeLanguages().length > 0,
  );
  public readonly hasPrivacy = computed(() => this.appConfig.getPrivacyLanguages().length > 0);
}
