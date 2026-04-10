/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useEffect, useState } from 'react';
import { Card, Spin } from '@douyinfe/semi-ui';
import SettingsGeneralPayment from '../../pages/Setting/Payment/SettingsGeneralPayment';
import SettingsPaymentGateway from '../../pages/Setting/Payment/SettingsPaymentGateway';
import SettingsPaymentGatewayLantu from '../../pages/Setting/Payment/SettingsPaymentGatewayLantu';
import SettingsPaymentGatewayStripe from '../../pages/Setting/Payment/SettingsPaymentGatewayStripe';
import SettingsPaymentGatewayCreem from '../../pages/Setting/Payment/SettingsPaymentGatewayCreem';
import SettingsPaymentGatewayWaffo from '../../pages/Setting/Payment/SettingsPaymentGatewayWaffo';
import { API, showError, toBoolean } from '../../helpers';
import { useTranslation } from 'react-i18next';

const PaymentSetting = () => {
  const { t } = useTranslation();
  let [inputs, setInputs] = useState({
    ServerAddress: '',
    PayAddress: '',
    EpayId: '',
    EpayKey: '',
    Price: 7.3,
    MinTopUp: 1,
    TopupGroupRatio: '',
    CustomCallbackAddress: '',
    PayMethods: '',
    AmountOptions: '',
    AmountBonus: '',
    AmountDiscount: '',

    StripeApiSecret: '',
    StripeWebhookSecret: '',
    StripePriceId: '',
    StripeUnitPrice: 8.0,
    StripeMinTopUp: 1,
    StripePromotionCodesEnabled: false,

    LantuApiUrl: '',
    LantuMchId: '',
    LantuSecretKey: '',
  });

  let [loading, setLoading] = useState(false);

  const getOptions = async () => {
    const res = await API.get('/api/option/');
    const { success, message, data } = res.data;
    if (success) {
      let newInputs = {};
      data.forEach((item) => {
        switch (item.key) {
          case 'TopupGroupRatio':
            try {
              newInputs[item.key] = JSON.stringify(
                JSON.parse(item.value),
                null,
                2,
              );
            } catch (error) {
              newInputs[item.key] = item.value;
            }
            break;
          case 'payment_setting.amount_options':
            try {
              newInputs['AmountOptions'] = JSON.stringify(
                JSON.parse(item.value),
                null,
                2,
              );
            } catch (error) {
              newInputs['AmountOptions'] = item.value;
            }
            break;
          case 'payment_setting.amount_bonus':
            try {
              newInputs['AmountBonus'] = JSON.stringify(
                JSON.parse(item.value),
                null,
                2,
              );
            } catch (error) {
              console.error('解析AmountBonus出错:', error);
              newInputs['AmountBonus'] = item.value;
            }
            break;
          case 'payment_setting.amount_discount':
            try {
              newInputs['AmountDiscount'] = JSON.stringify(
                JSON.parse(item.value),
                null,
                2,
              );
            } catch (error) {
              newInputs['AmountDiscount'] = item.value;
            }
            break;
          case 'Price':
          case 'MinTopUp':
          case 'StripeUnitPrice':
          case 'StripeMinTopUp':
            newInputs[item.key] = parseFloat(item.value);
            break;
          default:
            if (item.key.endsWith('Enabled')) {
              newInputs[item.key] = toBoolean(item.value);
            } else {
              newInputs[item.key] = item.value;
            }
            break;
        }
      });

      // Legacy compatibility:
      // If amount_bonus is not set but amount_discount looks like bonus rates (e.g. 0.05 ~ 0.4),
      // surface it in the UI as AmountBonus so admins can edit/migrate it.
      if (
        (!newInputs['AmountBonus'] || newInputs['AmountBonus'].trim() === '') &&
        typeof newInputs['AmountDiscount'] === 'string' &&
        newInputs['AmountDiscount'].trim() !== ''
      ) {
        try {
          const parsed = JSON.parse(newInputs['AmountDiscount']);
          const values = Object.values(parsed || {}).map((v) => Number(v));
          const looksLikeBonus =
            values.length > 0 &&
            values.every((v) => Number.isFinite(v) && v > 0 && v <= 0.5);
          if (looksLikeBonus) {
            newInputs['AmountBonus'] = newInputs['AmountDiscount'];
          }
        } catch (e) {
          // ignore
        }
      }

      setInputs(newInputs);
    } else {
      showError(t(message));
    }
  };

  async function onRefresh() {
    try {
      setLoading(true);
      await getOptions();
    } catch (error) {
      showError(t('刷新失败'));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    onRefresh();
  }, []);

  return (
    <>
      <Spin spinning={loading} size='large'>
        <Card style={{ marginTop: '10px' }}>
          <SettingsGeneralPayment options={inputs} refresh={onRefresh} />
        </Card>
        <Card style={{ marginTop: '10px' }}>
          <SettingsPaymentGateway options={inputs} refresh={onRefresh} />
        </Card>
        <Card style={{ marginTop: '10px' }}>
          <SettingsPaymentGatewayStripe options={inputs} refresh={onRefresh} />
        </Card>
        <Card style={{ marginTop: '10px' }}>
          <SettingsPaymentGatewayLantu options={inputs} refresh={onRefresh} />
        </Card>
        <Card style={{ marginTop: '10px' }}>
          <SettingsPaymentGatewayCreem options={inputs} refresh={onRefresh} />
        </Card>
        <Card style={{ marginTop: '10px' }}>
          <SettingsPaymentGatewayWaffo options={inputs} refresh={onRefresh} />
        </Card>
      </Spin>
    </>
  );
};

export default PaymentSetting;
