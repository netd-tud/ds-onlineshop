// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

using System;
using System.Threading.Tasks;
using Grpc.Core;
using Microsoft.Extensions.Logging;
using cartservice.cartstore;
using Hipstershop;
using cartservice.analytics;

namespace cartservice.services
{
    public class CartService : Hipstershop.CartService.CartServiceBase
    {
        private readonly static Empty Empty = new Empty();
        private readonly ICartStore _cartStore;
        private readonly AnalyticsPublisher _analyticsPublisher;

        public CartService(ICartStore cartStore, AnalyticsPublisher analyticsPublisher)
        {
            _cartStore = cartStore;
            _analyticsPublisher = analyticsPublisher;
        }

        public async override Task<Empty> AddItem(AddItemRequest request, ServerCallContext context)
        {
            var sessionId = context.RequestHeaders.GetValue("session-id") ?? string.Empty;

            await _cartStore.AddItemAsync(request.UserId, request.Item.ProductId, request.Item.Quantity);

            _analyticsPublisher.Publish(new ProductEvent {
                EventType = "ATC",
                Sku = request.Item.ProductId,
                Qty = request.Item.Quantity,
                SessionId = sessionId
            });
            return Empty;
        }

        public override Task<Cart> GetCart(GetCartRequest request, ServerCallContext context)
        {
            return _cartStore.GetCartAsync(request.UserId);
        }

        public async override Task<Empty> EmptyCart(EmptyCartRequest request, ServerCallContext context)
        {
            await _cartStore.EmptyCartAsync(request.UserId);
            return Empty;
        }
    }
}
