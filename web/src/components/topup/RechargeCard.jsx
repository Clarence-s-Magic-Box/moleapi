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

import React, { useEffect, useRef, useState } from 'react';
import {
  Avatar,
  Typography,
  Card,
  Button,
  Banner,
  Skeleton,
  Form,
  Space,
  Spin,
  Tooltip,
  Divider,
  Tabs,
  TabPane,
} from '@douyinfe/semi-ui';
import { SiAlipay, SiWechat, SiStripe } from 'react-icons/si';
import {
  CreditCard,
  Coins,
  Wallet,
  BarChart2,
  TrendingUp,
  Receipt,
  Sparkles,
  Gift,
} from 'lucide-react';
import { IconGift } from '@douyinfe/semi-icons';
import { useMinimumLoadingTime } from '../../hooks/common/useMinimumLoadingTime';
import { getCurrencyConfig } from '../../helpers/render';
import SubscriptionPlansCard from './SubscriptionPlansCard';

const { Text } = Typography;

const getPayMethodDisplayName = (payMethod, t) => {
  if (payMethod?.type === 'lantu') {
    return t('微信支付');
  }
  return payMethod?.name || '';
};

const RechargeCard = ({
  t,
  enableOnlineTopUp,
  enableStripeTopUp,
  enableCreemTopUp,
  creemProducts,
  creemPreTopUp,
  presetAmounts,
  selectedPreset,
  selectPresetAmount,
  formatLargeNumber,
  priceRatio,
  topUpCount,
  minTopUp,
  renderQuotaWithAmount,
  getAmount,
  setTopUpCount,
  setSelectedPreset,
  renderAmount,
  amountLoading,
  payMethods,
  preTopUp,
  paymentLoading,
  payWay,
  redemptionCode,
  setRedemptionCode,
  topUp,
  isSubmitting,
  topUpLink,
  openTopUpLink,
  userState,
  renderQuota,
  statusLoading,
  topupInfo,
  topupGroupRatio,
  onOpenHistory,
  subscriptionLoading = false,
  subscriptionPlans = [],
  billingPreference,
  onChangeBillingPreference,
  activeSubscriptions = [],
  allSubscriptions = [],
  reloadSubscriptionSelf,
  presetPayAmountMap = {},
}) => {
  const onlineFormApiRef = useRef(null);
  const redeemFormApiRef = useRef(null);
  const initialTabSetRef = useRef(false);
  const showAmountSkeleton = useMinimumLoadingTime(amountLoading);
  const [activeTab, setActiveTab] = useState('topup');
  const shouldShowSubscription =
    !subscriptionLoading && subscriptionPlans.length > 0;

  const getBonusRateForAmount = (amountValue) => {
    let bonusMap = topupInfo?.bonus || {};
    if (typeof bonusMap === 'string') {
      try {
        bonusMap = JSON.parse(bonusMap);
      } catch (e) {
        bonusMap = {};
      }
    }
    const amount = Number(amountValue);
    if (!Number.isFinite(amount)) return 0;

    const direct = bonusMap[amount];
    if (typeof direct === 'number' && direct > 0) return direct;

    const keys = Object.keys(bonusMap)
      .map((k) => Number(k))
      .filter((n) => Number.isFinite(n))
      .sort((a, b) => a - b);
    let best = 0;
    for (const k of keys) {
      if (k <= amount) {
        const v = Number(bonusMap[k]);
        if (Number.isFinite(v) && v > 0) best = v;
      }
    }
    return best;
  };

  const getExtraRatioText = (amountValue) => {
    const rate = getBonusRateForAmount(amountValue);
    if (!rate || rate <= 0) return t('无加赠');
    return `${t('加赠')}${Math.round(rate * 100)}%`;
  };

  const getUserGroupRatio = () => {
    // 普通用户无权读取 /api/option，倍率由 /api/user/topup/info 下发为“当前用户倍率”
    const n = Number(topupGroupRatio);
    return Number.isFinite(n) && n > 0 ? n : 1;
  };

  const calculateQuota = (amountValue) => {
    const amount = Number(amountValue);
    if (!Number.isFinite(amount) || amount <= 0) {
      return {
        baseQuota: 0,
        extraQuota: 0,
        totalQuota: 0,
        extraRatio: 0,
        groupRatio: getUserGroupRatio(),
      };
    }
    const baseQuota = amount * 500000; // 与 0.8.1 一致：1 USD = 500000 tokens
    const extraRatio = getBonusRateForAmount(amount);
    const groupRatio = getUserGroupRatio();
    const extraQuota = baseQuota * extraRatio;
    const finalBaseQuota = baseQuota * groupRatio;
    const finalExtraQuota = extraQuota * groupRatio;
    return {
      baseQuota: finalBaseQuota,
      extraQuota: finalExtraQuota,
      totalQuota: finalBaseQuota + finalExtraQuota,
      extraRatio,
      groupRatio,
    };
  };

  useEffect(() => {
    if (initialTabSetRef.current) return;
    if (subscriptionLoading) return;
    setActiveTab(shouldShowSubscription ? 'subscription' : 'topup');
    initialTabSetRef.current = true;
  }, [shouldShowSubscription, subscriptionLoading]);

  useEffect(() => {
    if (!shouldShowSubscription && activeTab !== 'topup') {
      setActiveTab('topup');
    }
  }, [shouldShowSubscription, activeTab]);
  const topupContent = (
    <Space vertical style={{ width: '100%' }}>
      {/* 统计数据 */}
      <Card
        className='!rounded-xl w-full'
        cover={
          <div
            className='relative h-30'
            style={{
              '--palette-primary-darkerChannel': '37 99 235',
              backgroundImage: `linear-gradient(0deg, rgba(var(--palette-primary-darkerChannel) / 80%), rgba(var(--palette-primary-darkerChannel) / 80%)), url('/cover-4.webp')`,
              backgroundSize: 'cover',
              backgroundPosition: 'center',
              backgroundRepeat: 'no-repeat',
            }}
          >
            <div className='relative z-10 h-full flex flex-col justify-between p-4'>
              <div className='flex justify-between items-center'>
                <Text strong style={{ color: 'white', fontSize: '16px' }}>
                  {t('账户统计')}
                </Text>
              </div>

              {/* 统计数据 */}
              <div className='grid grid-cols-3 gap-6 mt-4'>
                {/* 当前余额 */}
                <div className='text-center'>
                  <div
                    className='text-base sm:text-2xl font-bold mb-2'
                    style={{ color: 'white' }}
                  >
                    {renderQuota(userState?.user?.quota)}
                  </div>
                  <div className='flex items-center justify-center text-sm'>
                    <Wallet
                      size={14}
                      className='mr-1'
                      style={{ color: 'rgba(255,255,255,0.8)' }}
                    />
                    <Text
                      style={{
                        color: 'rgba(255,255,255,0.8)',
                        fontSize: '12px',
                      }}
                    >
                      {t('当前余额')}
                    </Text>
                  </div>
                </div>

                {/* 历史消耗 */}
                <div className='text-center'>
                  <div
                    className='text-base sm:text-2xl font-bold mb-2'
                    style={{ color: 'white' }}
                  >
                    {renderQuota(userState?.user?.used_quota)}
                  </div>
                  <div className='flex items-center justify-center text-sm'>
                    <TrendingUp
                      size={14}
                      className='mr-1'
                      style={{ color: 'rgba(255,255,255,0.8)' }}
                    />
                    <Text
                      style={{
                        color: 'rgba(255,255,255,0.8)',
                        fontSize: '12px',
                      }}
                    >
                      {t('历史消耗')}
                    </Text>
                  </div>
                </div>

                {/* 请求次数 */}
                <div className='text-center'>
                  <div
                    className='text-base sm:text-2xl font-bold mb-2'
                    style={{ color: 'white' }}
                  >
                    {userState?.user?.request_count || 0}
                  </div>
                  <div className='flex items-center justify-center text-sm'>
                    <BarChart2
                      size={14}
                      className='mr-1'
                      style={{ color: 'rgba(255,255,255,0.8)' }}
                    />
                    <Text
                      style={{
                        color: 'rgba(255,255,255,0.8)',
                        fontSize: '12px',
                      }}
                    >
                      {t('请求次数')}
                    </Text>
                  </div>
                </div>
              </div>
            </div>
          </div>
        }
      >
        {/* 在线充值表单 */}
        {statusLoading ? (
          <div className='py-8 flex justify-center'>
            <Spin size='large' />
          </div>
        ) : enableOnlineTopUp || enableStripeTopUp || enableCreemTopUp ? (
          <Form
            getFormApi={(api) => (onlineFormApiRef.current = api)}
            initValues={{ topUpCount: topUpCount }}
          >
            <div className='space-y-4'>
              {/* ① 选择充值额度 - 预设卡片在最上方 */}
              {(enableOnlineTopUp || enableStripeTopUp) && (
                <Form.Slot
                  label={
                    <div className='flex items-center gap-2'>
                      <span
                        style={{
                          color: 'var(--semi-color-primary)',
                          fontWeight: 600,
                        }}
                      >
                        {t('选择充值额度')}
                      </span>
                      {(() => {
                        const { symbol, rate, type } = getCurrencyConfig();
                        if (type === 'USD') return null;
                        return (
                          <span style={{ color: 'var(--semi-color-text-2)', fontSize: '12px', fontWeight: 'normal' }}>
                            (1 $ = {rate.toFixed(2)} {symbol})
                          </span>
                        );
                      })()}
                    </div>
                  }
                >
                  <div className='grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-3 p-1 overflow-visible'>
                    {presetAmounts.map((preset, index) => {
                      const quotaInfo = calculateQuota(preset.value);
                      const isSelected = selectedPreset === preset.value;
                      const presetPayAmount = Number(
                        presetPayAmountMap?.[preset.value],
                      );
                      const payAmountText = Number.isFinite(presetPayAmount)
                        ? presetPayAmount.toFixed(2)
                        : '--';
                      return (
                        <Card
                          key={index}
                          onClick={() => {
                            selectPresetAmount(preset);
                            onlineFormApiRef.current?.setValue(
                              'topUpCount',
                              preset.value,
                            );
                          }}
                              className={`cursor-pointer !rounded-2xl transition-all duration-200 hover:shadow-md ${
                                isSelected
                                  ? 'border-2 border-blue-500 shadow-lg'
                                  : 'border border-gray-200 hover:border-blue-300 hover:bg-gray-50'
                              }`}
                              style={{
                                transform: isSelected ? 'scale(1.02)' : 'scale(1)',
                                transformOrigin: 'center',
                                zIndex: isSelected ? 2 : 1,
                                backgroundColor: isSelected
                                  ? 'rgba(59, 130, 246, 0.14)'
                                  : undefined,
                              }}
                            bodyStyle={{
                              textAlign: 'center',
                              padding: '12px',
                              position: 'relative',
                              backgroundColor: isSelected
                                ? 'rgba(59, 130, 246, 0.14)'
                                : undefined,
                            }}
                          >
                          {isSelected && (
                            <div className='absolute top-2 right-2'>
                              <div className='w-5 h-5 bg-blue-500 rounded-full flex items-center justify-center'>
                                <span className='text-white text-xs'>✓</span>
                              </div>
                            </div>
                          )}
                          <div
                            className={`font-bold text-lg flex items-center justify-center mb-1 ${
                              isSelected ? 'text-blue-700' : ''
                            }`}
                          >
                            <Coins
                              size={16}
                              className={`mr-0.5 ${
                                isSelected ? 'text-blue-600' : ''
                              }`}
                            />
                            ${formatLargeNumber(preset.value)}
                          </div>
                          <div
                            className={`text-xs mb-1 ${
                              isSelected ? 'text-blue-600' : 'text-gray-500'
                            }`}
                          >
                              {t('实付')} ￥
                              {payAmountText}
                            </div>
                          <div
                            className={`text-xs font-medium ${
                              isSelected ? 'text-green-700' : 'text-green-600'
                            }`}
                          >
                            {getExtraRatioText(preset.value)}
                          </div>
                          <div
                            className={`text-xs mt-1 ${
                              isSelected ? 'text-blue-500' : 'text-gray-400'
                            }`}
                          >
                            {t('实到')} {renderQuota(quotaInfo.totalQuota)}
                          </div>
                        </Card>
                      );
                    })}
                  </div>
                </Form.Slot>
              )}

              {/* ② 分割线 - 或输入自定义金额 */}
              {(enableOnlineTopUp || enableStripeTopUp) && (
                <Divider margin={12}>
                  <Text type='tertiary' style={{ fontSize: '13px' }}>{t('或输入自定义金额')}</Text>
                </Divider>
              )}

              {/* ③ 充值数量 + 实付信息 */}
              {(enableOnlineTopUp || enableStripeTopUp) && (
                <div>
                  <div className='flex items-center justify-between mb-2'>
                    <Text strong style={{ fontSize: '14px' }}>{t('充值数量')}</Text>
                    <Skeleton
                      loading={showAmountSkeleton}
                      active
                      placeholder={<Skeleton.Title style={{ width: 120, height: 18, borderRadius: 6 }} />}
                    >
                      <div className='text-right'>
                        <Text style={{ fontSize: '13px' }}>
                          {t('实付金额：')}{renderAmount()}
                        </Text>
                        {(() => {
                          const quotaInfo = calculateQuota(topUpCount);
                          if (quotaInfo.extraRatio > 0) {
                            return (
                              <div>
                                <Text
                                  type='success'
                                  style={{ fontSize: '12px', fontWeight: 600 }}
                                >
                                  {getExtraRatioText(topUpCount)} • {t('总计')}{' '}
                                  {renderQuota(quotaInfo.totalQuota)}
                                </Text>
                              </div>
                            );
                          }
                          return null;
                        })()}
                      </div>
                    </Skeleton>
                  </div>
                  <Form.InputNumber
                    field='topUpCount'
                    noLabel={true}
                    disabled={!enableOnlineTopUp && !enableStripeTopUp}
                    placeholder={t('充值数量，最低 ') + renderQuotaWithAmount(minTopUp)}
                    value={topUpCount}
                    min={minTopUp}
                    max={999999999}
                    step={1}
                    precision={0}
                    onChange={async (value) => {
                      if (value && value >= 1) {
                        setTopUpCount(value);
                        setSelectedPreset(null);
                        await getAmount(value);
                      }
                    }}
                    onBlur={(e) => {
                      const value = parseInt(e.target.value);
                      if (!value || value < 1) {
                        setTopUpCount(1);
                        getAmount(1);
                      }
                    }}
                    formatter={(value) => (value ? `${value}` : '')}
                    parser={(value) => value ? parseInt(value.replace(/[^\d]/g, '')) : 0}
                    style={{ width: '100%' }}
                  />
                </div>
              )}

              {/* ④ 选择支付方式 */}
              {(enableOnlineTopUp || enableStripeTopUp) && (
                <Form.Slot label={<Text strong style={{ fontSize: '14px' }}>{t('选择支付方式')}</Text>}>
                  {payMethods && payMethods.length > 0 ? (
                    <div className='flex flex-col gap-2'>
                      {payMethods.map((payMethod) => {
                        const minTopupVal = Number(payMethod.min_topup) || 0;
                        const isStripe = payMethod.type === 'stripe';
                        const disabled =
                          (!enableOnlineTopUp && !isStripe) ||
                          (!enableStripeTopUp && isStripe) ||
                          minTopupVal > Number(topUpCount || 0);

                        // 根据支付类型决定按钮颜色
                        const getPayBtnStyle = (type) => {
                          if (type === 'wxpay') return { background: '#07C160', borderColor: '#07C160' };
                          if (type === 'alipay') return { background: '#1677FF', borderColor: '#1677FF' };
                          if (type === 'stripe') return { background: '#635BFF', borderColor: '#635BFF' };
                          if (type === 'lantu') return { background: '#07C160', borderColor: '#07C160' };
                          return {};
                        };

                        const buttonEl = (
                          <Button
                            key={payMethod.type}
                            theme='solid'
                            onClick={() => preTopUp(payMethod.type)}
                            disabled={disabled}
                            loading={paymentLoading && payWay === payMethod.type}
                            icon={
                              payMethod.type === 'alipay' ? (
                                <SiAlipay size={18} />
                              ) : payMethod.type === 'wxpay' ? (
                                <SiWechat size={18} />
                              ) : payMethod.type === 'stripe' ? (
                                <SiStripe size={18} />
                              ) : payMethod.type === 'lantu' ? (
                                <SiWechat size={18} />
                              ) : (
                                <CreditCard size={18} />
                              )
                            }
                            style={{
                              width: '100%',
                              height: '44px',
                              borderRadius: '10px',
                              fontSize: '15px',
                              fontWeight: 600,
                              ...(!disabled ? getPayBtnStyle(payMethod.type) : {}),
                            }}
                          >
                            {getPayMethodDisplayName(payMethod, t)}
                          </Button>
                        );

                        return disabled && minTopupVal > Number(topUpCount || 0) ? (
                          <Tooltip
                            content={t('此支付方式最低充值金额为') + ' ' + minTopupVal}
                            key={payMethod.type}
                          >
                            {buttonEl}
                          </Tooltip>
                        ) : (
                          <React.Fragment key={payMethod.type}>{buttonEl}</React.Fragment>
                        );
                      })}
                    </div>
                  ) : (
                    <div className='text-gray-500 text-sm p-3 bg-gray-50 rounded-lg border border-dashed border-gray-300'>
                      {t('暂无可用的支付方式，请联系管理员配置')}
                    </div>
                  )}
                </Form.Slot>
              )}

              {/* 加赠方案说明（放在支付按钮下方） */}
              {(enableOnlineTopUp || enableStripeTopUp) &&
                topupInfo?.bonus &&
                Object.keys(topupInfo.bonus).length > 0 && (
                  <div className='mt-4 mb-2'>
                    <Card className='!rounded-2xl bg-gradient-to-r from-blue-50 to-purple-50'>
                      <div className='flex items-start mb-4'>
                        <Gift
                          size={16}
                          className='mr-2 mt-0.5 text-purple-600'
                        />
                        <Text strong className='text-purple-800'>
                          {t('充值加赠方案')}
                        </Text>
                      </div>
                      <div className='grid grid-cols-2 md:grid-cols-4 gap-3 text-xs mb-4'>
                        {presetAmounts.map((preset, index) => (
                          <div
                            key={index}
                            className='text-center p-3 bg-white rounded-lg shadow-sm'
                          >
                            <div className='font-medium text-sm'>
                              ${preset.value}+
                            </div>
                            <div className='text-green-600 font-medium'>
                              {getExtraRatioText(preset.value)}
                            </div>
                          </div>
                        ))}
                      </div>
                      <Text type='tertiary' className='text-xs block text-center'>
                        {t(
                          '充值金额越高，加赠比例越多，最高可获得40%额外奖励',
                        )}
                      </Text>
                      <Text
                        type='tertiary'
                        className='text-xs block text-center mt-2'
                      >
                        {t('充值支持按实付金额开票')}
                      </Text>
                    </Card>
                  </div>
                )}

              {/* Creem 充值区域 */}
              {enableCreemTopUp && creemProducts.length > 0 && (
                <Form.Slot label={t('Creem 充值')}>
                  <div className='grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-3'>
                    {creemProducts.map((product, index) => (
                      <Card
                        key={index}
                        onClick={() => creemPreTopUp(product)}
                        className='cursor-pointer !rounded-2xl transition-all hover:shadow-md border-gray-200 hover:border-gray-300'
                        bodyStyle={{ textAlign: 'center', padding: '16px' }}
                      >
                        <div className='font-medium text-lg mb-2'>
                          {product.name}
                        </div>
                        <div className='text-sm text-gray-600 mb-2'>
                          {t('充值额度')}: {product.quota}
                        </div>
                        <div className='text-lg font-semibold text-blue-600'>
                          {product.currency === 'EUR' ? '€' : '$'}
                          {product.price}
                        </div>
                      </Card>
                    ))}
                  </div>
                </Form.Slot>
              )}
            </div>
          </Form>
        ) : (
          <Banner
            type='info'
            description={t(
              '管理员未开启在线充值功能，请联系管理员开启或使用兑换码充值。',
            )}
            className='!rounded-xl'
            closeIcon={null}
          />
        )}
      </Card>

      {/* 兑换码充值 */}
      <Card
        className='!rounded-xl w-full'
        title={
          <Text type='tertiary' strong>
            {t('兑换码充值')}
          </Text>
        }
      >
        <Form
          getFormApi={(api) => (redeemFormApiRef.current = api)}
          initValues={{ redemptionCode: redemptionCode }}
        >
          <Form.Input
            field='redemptionCode'
            noLabel={true}
            placeholder={t('请输入兑换码')}
            value={redemptionCode}
            onChange={(value) => setRedemptionCode(value)}
            prefix={<IconGift />}
            suffix={
              <div className='flex items-center gap-2'>
                <Button
                  type='primary'
                  theme='solid'
                  onClick={topUp}
                  loading={isSubmitting}
                >
                  {t('兑换额度')}
                </Button>
              </div>
            }
            showClear
            style={{ width: '100%' }}
            extraText={
              topUpLink && (
                <Text type='tertiary'>
                  {t('在找兑换码？')}
                  <Text
                    type='secondary'
                    underline
                    className='cursor-pointer'
                    onClick={openTopUpLink}
                  >
                    {t('购买兑换码')}
                  </Text>
                </Text>
              )
            }
          />
        </Form>
      </Card>
    </Space>
  );

  return (
    <Card className='!rounded-2xl shadow-sm border-0'>
      {/* 卡片头部 */}
      <div className='flex items-center justify-between mb-4'>
        <div className='flex items-center'>
          <Avatar size='small' color='blue' className='mr-3 shadow-md'>
            <CreditCard size={16} />
          </Avatar>
          <div>
            <Typography.Text className='text-lg font-medium'>
              {t('账户充值')}
            </Typography.Text>
            <div className='text-xs'>{t('多种充值方式，安全便捷')}</div>
          </div>
        </div>
        <Button
          icon={<Receipt size={16} />}
          theme='solid'
          onClick={onOpenHistory}
        >
          {t('账单')}
        </Button>
      </div>

      {shouldShowSubscription ? (
        <Tabs type='card' activeKey={activeTab} onChange={setActiveTab}>
          <TabPane
            tab={
              <div className='flex items-center gap-2'>
                <Sparkles size={16} />
                {t('订阅套餐')}
              </div>
            }
            itemKey='subscription'
          >
            <div className='py-2'>
              <SubscriptionPlansCard
                t={t}
                loading={subscriptionLoading}
                plans={subscriptionPlans}
                payMethods={payMethods}
                enableOnlineTopUp={enableOnlineTopUp}
                enableStripeTopUp={enableStripeTopUp}
                enableCreemTopUp={enableCreemTopUp}
                billingPreference={billingPreference}
                onChangeBillingPreference={onChangeBillingPreference}
                activeSubscriptions={activeSubscriptions}
                allSubscriptions={allSubscriptions}
                reloadSubscriptionSelf={reloadSubscriptionSelf}
                withCard={false}
              />
            </div>
          </TabPane>
          <TabPane
            tab={
              <div className='flex items-center gap-2'>
                <Wallet size={16} />
                {t('额度充值')}
              </div>
            }
            itemKey='topup'
          >
            <div className='py-2'>{topupContent}</div>
          </TabPane>
        </Tabs>
      ) : (
        topupContent
      )}
    </Card>
  );
};

export default RechargeCard;
