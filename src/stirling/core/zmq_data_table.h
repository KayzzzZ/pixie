//
// Created by 千陆 on 2022/4/13.
//

#ifndef PIXIE_ZMQ_DATA_TABLE_H
#define PIXIE_ZMQ_DATA_TABLE_H

#include <algorithm>
#include <string>
#include <utility>
#include <vector>

#include "src/common/base/base.h"
#include "src/shared/types/type_utils.h"
#include "src/stirling/core/data_table.h"
#include "src/stirling/core/types.h"
#include "src/stirling/utils/index_sorted_vector.h"

namespace px {
namespace stirling {

class ZmqDataTable : public DataTable {
public:
    // Global unique ID identifies the table store to which this DataTable's data should be pushed.
    ZmqDataTable(uint64_t id, const DataTableSchema& schema);
    virtual ~ZmqDataTable() = default;

    std::vector<TaggedRecordBatch> ConsumeRecords() override;

    double OccupancyPct() const { return 1.0 * Occupancy() / kTargetCapacity; }

protected:
    // Initialize a new Active record batch.
    void InitBuffers(types::ColumnWrapperRecordBatch* record_batch_ptr) override;

    // Get a pointer to the Tablet, for appending. Used by RecordBuilder.
    Tablet* GetTablet(types::TabletIDView tablet_id) override;

};
}
}


#endif //PIXIE_ZMQ_DATA_TABLE_H
