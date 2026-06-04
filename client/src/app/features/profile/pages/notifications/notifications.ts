import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { ReactiveFormsModule } from '@angular/forms';
import { TranslatePipe } from '@ngx-translate/core';
import { SwPush } from '@angular/service-worker';
import { AppConfig } from '../../../../core/services/app-config';
import { ProfileStore } from '../../services/profile-store';

@Component({
  selector: 'app-profile-infos',
  imports: [ReactiveFormsModule, TranslatePipe],
  templateUrl: './notifications.html',
  styleUrl: './notifications.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ProfileNotificationsPage {
  private appConfig = inject(AppConfig);
  private swPush = inject(SwPush);

  public store = inject(ProfileStore);

  public requestWebpush(): void {
    const serverPublicKey = this.appConfig.getAppInfo()()?.webpush_public_key;
    if (!serverPublicKey) {
      return;
    }

    this.swPush.requestSubscription({ serverPublicKey }).then((m) => console.log(JSON.stringify(m)));
  }
}
