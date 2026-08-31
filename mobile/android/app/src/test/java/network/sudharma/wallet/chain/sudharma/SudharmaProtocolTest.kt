package network.sudharma.wallet.chain.sudharma

import com.squareup.moshi.Json
import com.squareup.moshi.Moshi
import com.squareup.moshi.kotlin.reflect.KotlinJsonAdapterFactory
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import java.math.BigInteger

class SudharmaProtocolTest {
    private data class GoldenVector(
        val version: Int,
        @Json(name = "private_scalar_hex") val privateScalarHex: String,
        @Json(name = "public_key_hex") val publicKeyHex: String,
        val address: String,
        val recipient: String,
        val amount: Long,
        val fee: Long,
        val nonce: Long,
        @Json(name = "transaction_id") val transactionId: String,
        @Json(name = "signature_hex") val signatureHex: String,
    )

    private fun goldenVector(): GoldenVector {
        val json = requireNotNull(
            javaClass.classLoader?.getResourceAsStream("android_wallet_protocol_v1.json"),
        ) { "shared Android wallet golden vector is missing" }
            .bufferedReader()
            .use { it.readText() }
        return requireNotNull(
            Moshi.Builder()
                .add(KotlinJsonAdapterFactory())
                .build()
                .adapter(GoldenVector::class.java)
                .fromJson(json),
        ) { "shared Android wallet golden vector is invalid" }
    }

    @Test
    fun sharedGoldenVectorMatchesGoAddressAndTransactionRules() {
        val vector = goldenVector()
        assertEquals(1, vector.version)
        val key = SudharmaCrypto.keyFromPrivateScalar(BigInteger(vector.privateScalarHex, 16))
        assertEquals(vector.publicKeyHex, key.publicKey.toHex())
        assertEquals(vector.address, SudharmaCrypto.addressFromPublicKey(key.publicKey))
        val tx = SudharmaTransaction.create(
            from = vector.address,
            to = vector.recipient,
            amount = vector.amount,
            nonce = vector.nonce,
        )
        assertEquals(vector.fee, tx.fee)
        assertEquals(vector.transactionId, tx.id)
        val signature = vector.signatureHex.hexToBytes()
        assertEquals(64, signature.size)
        assertTrue(SudharmaCrypto.verify(key.publicKey, tx.id.toByteArray(), signature))
    }

    @Test
    fun localSignatureUsesFixedWidthAndVerifies() {
        val key = SudharmaCrypto.keyFromPrivateScalar(BigInteger.ONE)
        val tx = SudharmaTransaction.create(
            from = SudharmaCrypto.addressFromPublicKey(key.publicKey),
            to = "00112233445566778899aabbccddeeff00112233",
            amount = 500_000_000L,
            nonce = 1L,
        )
        val signature = SudharmaCrypto.sign(key.privateScalar, tx.id.toByteArray())
        assertEquals(64, signature.size)
        assertTrue(SudharmaCrypto.verify(key.publicKey, tx.id.toByteArray(), signature))
    }

    private fun ByteArray.toHex(): String = joinToString("") { "%02x".format(it.toInt() and 0xff) }

    private fun String.hexToBytes(): ByteArray {
        require(length % 2 == 0)
        return chunked(2).map { it.toInt(16).toByte() }.toByteArray()
    }
}
