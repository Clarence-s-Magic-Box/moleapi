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

import React, { useEffect, useState, useRef } from 'react';
import {
  Button,
  Form,
  Row,
  Col,
  Typography,
  Spin,
} from '@douyinfe/semi-ui';
const { Text } = Typography;
import {
  API,
  showError,
  showSuccess,
} from '../../../helpers';
import { useTranslation } from 'react-i18next';

export default function SettingsPaymentGatewayLantu(props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState({
    LantuMchId: '',
    LantuSecretKey: '',
  });
  const formApiRef = useRef(null);

  useEffect(() => {
    if (props.options && formApiRef.current) {
      const currentInputs = {
        LantuMchId: props.options.LantuMchId || '',
        LantuSecretKey: props.options.LantuSecretKey || '',
      };
      setInputs(currentInputs);
      formApiRef.current.setValues(currentInputs);
    }
  }, [props.options]);

  const handleFormChange = (values) => {
    setInputs(values);
  };

  const submitLantuSettings = async () => {
    if (props.options.ServerAddress === '') {
      showError(t('请先填写服务器地址'));
      return;
    }

    setLoading(true);
    try {
      const options = [];

      if (inputs.LantuMchId !== '') {
        options.push({ key: 'LantuMchId', value: inputs.LantuMchId });
      }
      if (inputs.LantuSecretKey !== '') {
        options.push({ key: 'LantuSecretKey', value: inputs.LantuSecretKey });
      }

      // 发送请求
      const requestQueue = options.map(opt =>
        API.put('/api/option/', {
          key: opt.key,
          value: opt.value,
        })
      );

      const results = await Promise.all(requestQueue);

      // 检查所有请求是否成功
      const errorResults = results.filter(res => !res.data.success);
      if (errorResults.length > 0) {
        errorResults.forEach(res => {
          showError(res.data.message);
        });
      } else {
        showSuccess(t('更新成功'));
        props.refresh && props.refresh();
      }
    } catch (error) {
      showError(t('更新失败'));
    }
    setLoading(false);
  };

  return (
    <Spin spinning={loading}>
      <Form
        initValues={inputs}
        onValueChange={handleFormChange}
        getFormApi={(api) => (formApiRef.current = api)}
      >
        <Form.Section text={t('蓝兔支付设置')}>
          <Text>
            {t('（仅需配置商户信息；回调地址使用“服务器地址/自定义回调地址”设置）')}
          </Text>
          <Row
            gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
          >
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='LantuMchId'
                label={t('蓝兔支付商户ID')}
                placeholder={t('请输入商户ID')}
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='LantuSecretKey'
                label={t('蓝兔支付密钥')}
                placeholder={t('敏感信息不会发送到前端显示')}
                type='password'
              />
            </Col>
          </Row>

          <Button onClick={submitLantuSettings}>{t('更新蓝兔支付设置')}</Button>
        </Form.Section>
      </Form>
    </Spin>
  );
}
